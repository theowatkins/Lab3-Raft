package main

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

/* On Win Handler. Responsibilities includes
 * - updating role of server to leader
 * - running heartbeat thread
 * - managing client requests (in progress)
 */
func onWinChannelListener(
	leaderServerState *ServerState,
	onWinChannel *chan bool,
	serverStateLock *sync.Mutex,
	appendEntriesCom *[8]AppendEntriesCom,
	clientCommunicationChannel *chan KeyValue,
	persister Persister,
	channel ApplyChannel,
	) {

	for {
		select {
		case <-* onWinChannel: // got enough votes
			serverStateLock.Lock()
			leaderServerState.Role = LeaderRole
			serverStateLock.Unlock()

			// initialize leader leaderServerState
			var leaderState [ClusterSize]ServerTermState
			for i := 0; i < ClusterSize; i++ {
				// Note that logentries are indexed from 1
				leaderState[i] = ServerTermState{len(leaderServerState.Log) + 1, 0}
			}

			go runHeartbeatThread(leaderServerState, appendEntriesCom) // Implements L1.
			go readAndDistributeClientRequests(leaderServerState, &leaderState, appendEntriesCom, clientCommunicationChannel, persister, channel)
		}
	}
}

func runHeartbeatThread(
	leaderServerState *ServerState,
	appendEntriesCom *[ClusterSize]AppendEntriesCom) {
	for leaderServerState.Role == LeaderRole {
		for _, leaderCommunicationChannel := range *appendEntriesCom {
			leaderCommunicationChannel.message <- AppendEntriesMessage{
				// leader's term
				leaderServerState.CurrentTerm,

				// leader's ID
				leaderServerState.ServerId,

				// index of last entry in log
				len(leaderServerState.Log),

				// term of last entry in log
				leaderServerState.CurrentTerm,

				// list of logentries to store
				// ** empty for heartbeat **
				[]LogEntry{},

				// leader's current commit index
				leaderServerState.commitIndex}
		}
		time.Sleep(time.Duration(HeartBeatDelay) * time.Millisecond)
	}
}


func readAndDistributeClientRequests(
	leaderServerState *ServerState,
	serverLeaderStates *[ClusterSize]ServerTermState,
	appendEntriesCom *[ClusterSize]AppendEntriesCom,
	clientCommunicationChannel *chan KeyValue,
	persister Persister,
	channel ApplyChannel,
	) {
	/* AppendEntriesRequest Handler
	 * Distributes a client request to all servers in the system.
	 */
	go func () {
		for leaderServerState.Role == LeaderRole {
			// implements L4
			n := 0
			count := 0

			// get smallest N greater than commitIndex
			for _, curState := range serverLeaderStates {
				if curState.matchIndex > leaderServerState.commitIndex {
					if curState.matchIndex < n || n == 0 {
						n = curState.matchIndex
					}
				}
			}

			if n != 0 {
				for _, curState := range serverLeaderStates {
					if curState.matchIndex >= n {
						count++
					}
				}
			}

			// on majority of matchIndex >= n, update commitIndex
			if count > ClusterSize / 2 && leaderServerState.Log[n - 1].Term == leaderServerState.CurrentTerm {
				leaderServerState.commitIndex = n
			} // end implementation of L4

			select {
			case clientRequest := <-*clientCommunicationChannel:

				clientLogEntry := LogEntry{leaderServerState.CurrentTerm, clientRequest}

				leaderServerState.Log = append(leaderServerState.Log, clientLogEntry)
				
				for serverIndex, _ := range *appendEntriesCom {
					go sendAppendEntriesMessage(
						serverIndex,
						appendEntriesCom, 
						serverLeaderStates, 
						leaderServerState)
				}

				err := persister.Save(strconv.Itoa(leaderServerState.commitIndex), clientLogEntry)
				for err != nil { //retry until no error.
					err = persister.Save(strconv.Itoa(leaderServerState.commitIndex), clientLogEntry)
				}

				go func() { //send when channel is ready
					channel <- ApplyMessage {
						term:     leaderServerState.CurrentTerm,
						logEntry: clientLogEntry,
					}
				}()
			}
		}
	}()

	/* AppendEntriesResponse Handlers
	 * For each server a goroutine is created that continuously reads AppendEntries resopnses
	 */
	go func(){
		for serverIndex, serverAppendEntriesCom := range *appendEntriesCom {
			serverAppendEntriesCom := serverAppendEntriesCom
			serverIndex := serverIndex
			go func() {
				for leaderServerState.Role == LeaderRole {
					select {
					case r := <-serverAppendEntriesCom.response:
						if r.term > leaderServerState.CurrentTerm {
							staleTerm(leaderServerState, r.term)
						} else {
							if r.success {
								// implements L3
								curMatchIndex := r.message.PrevLogIndex + len(r.message.Entries)
								serverLeaderStates[serverIndex].matchIndex = curMatchIndex
								serverLeaderStates[serverIndex].nextIndex = curMatchIndex + 1
							} else {
								fmt.Println("Server ", serverIndex, " had error processing AppendEntries request: ", r.message)
								// resend message with more logEntries on failure
								if serverLeaderStates[serverIndex].nextIndex > 1 {
									serverLeaderStates[serverIndex].nextIndex -= 1 //
								}

								go sendAppendEntriesMessage(
									serverIndex,
									appendEntriesCom, 
									serverLeaderStates, 
									leaderServerState)
							}
						}
					}
				}
				fmt.Print("Leader stopped being a leader...\n")
			}()
		}
	}()

	fmt.Println("Client listeners where launched successfully.")
}

func sendAppendEntriesMessage(
	serverIndex int,
	appendEntriesCom *[ClusterSize]AppendEntriesCom,
	leaderStates *[ClusterSize]ServerTermState,
	leaderServerState * ServerState) {
	
	curLeaderState := leaderStates[serverIndex]

	if len(leaderServerState.Log) >= curLeaderState.nextIndex {// implements L3
		entries := leaderServerState.Log[curLeaderState.nextIndex - 1:]
		prevLogIndex := curLeaderState.nextIndex-1
		prevLogTerm := -1
		if prevLogIndex > 0 && prevLogIndex < len(leaderServerState.Log) {
			prevLogTerm = leaderServerState.Log[prevLogIndex - 1].Term
		}
		appendEntriesCom[serverIndex].message <- AppendEntriesMessage {
			// leader's term
			leaderServerState.CurrentTerm,

			// leader's ID
			leaderServerState.ServerId,

			// index of previous entry in log
			prevLogIndex,

			// term of previous entry in log
			prevLogTerm,

			// new LogEntries to store
			// from nextIndex to end of log
			entries,

			// leader's current commit index
			leaderServerState.commitIndex}
	}
}

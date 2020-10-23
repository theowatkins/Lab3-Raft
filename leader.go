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
	//TODO: What if received message from new leader before onWinChannel gets to hear about it?
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

//TODO: update commitindex on majority
func readAndDistributeClientRequests(
	leaderServerState *ServerState,
	serverLeaderStates *[ClusterSize]ServerTermState,
	appendEntriesCom *[ClusterSize]AppendEntriesCom,
	clientCommunicationChannel *chan KeyValue,
	persister Persister,
	channel ApplyChannel,
	) {

	censusedReachedChannel := make(chan bool)
	nodesWithReplicatedEntry := 0
	var currentLogEntry LogEntry
	/* AppendEntriesRequest Handler
	 * Distributes a client request to all servers in the system.
	 */
	go func () {
		for leaderServerState.Role == LeaderRole {
			select {
			case clientRequest := <-*clientCommunicationChannel:

				clientLogEntry := LogEntry{leaderServerState.CurrentTerm, clientRequest}
				currentLogEntry = clientLogEntry

				leaderServerState.Log = append(leaderServerState.Log, clientLogEntry)
				leaderServerState.lastApplied++

				// implements L4
				n := 0
				count := 0

				// get smallest N greater than commitIndex
				for _, curState := range serverLeaderStates {
					if curState.matchIndex > leaderServerState.commitIndex {
						if n == 0 || curState.matchIndex < n {
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
				if count > ClusterSize / 2 && leaderServerState.Log[n].Term == leaderServerState.CurrentTerm {
					leaderServerState.commitIndex = n
				} // end implementation of L4

				
				for serverIndex, leaderCommunicationChannel := range *appendEntriesCom {
					
					// implements L3
					if len(leaderServerState.Log) >= serverLeaderStates[serverIndex].nextIndex {
						go sendAppendEntriesMessage(
							leaderCommunicationChannel,
							[]LogEntry{clientLogEntry},
							leaderServerState,
							len(leaderServerState.Log),
							leaderServerState.CurrentTerm,
							)
					}
				}

				<- censusedReachedChannel

				err := persister.Save(strconv.Itoa(len(leaderServerState.Log)), clientLogEntry)
				for err != nil { //retry until no error.
					err = persister.Save(strconv.Itoa(len(leaderServerState.Log)), clientLogEntry)
				}

				leaderServerState.commitIndex++
				nodesWithReplicatedEntry = 0 //clear count for next client requests
				channel <- ApplyMessage {
					term:     leaderServerState.CurrentTerm,
					logEntry: clientLogEntry,
				}
			}
		}
	}()

	/* AppendEntriesResponse Handlers
	 * For each server a goroutine is created that continuously reads AppendEntries resopnses
	 */
	go func(){
		newClientRequest := false
		for serverIndex, serverAppendEntriesCom := range *appendEntriesCom {
			serverAppendEntriesCom := serverAppendEntriesCom
			serverIndex := serverIndex
			go func() {
				for leaderServerState.Role == LeaderRole {
					select {
					case r := <-serverAppendEntriesCom.response:
						// TODO: why does the paper say to respond with a term?!
						if r.success {
							// implements L3
							serverLeaderStates[serverIndex].matchIndex = r.message.PrevLogIndex + len(r.message.Entries)
							serverLeaderStates[serverIndex].nextIndex = serverLeaderStates[serverIndex].matchIndex + 1


							/* nodesWithReplicatedEntry should only update if it is towards the majority.
							 * Once reached it is clear by the client handler. Therefore, we not continue to update
							 * it once the majority is reached
							 */
							if nodesWithReplicatedEntry >= ClusterSize / 2 && newClientRequest {
								censusedReachedChannel <- true
								newClientRequest = false
							} else {
								if nodesWithReplicatedEntry == 0 {
									newClientRequest = true
								}
								nodesWithReplicatedEntry++
							}
						} else {
							fmt.Println("Server ", serverIndex, " had error processing AppendEntries request: ", r.message)
							// resend message with more logEntries on failure
							if serverLeaderStates[serverIndex].nextIndex > 1 {
								serverLeaderStates[serverIndex].nextIndex -= 1 //
							}

							//implements L3
							if len(leaderServerState.Log) >= serverLeaderStates[serverIndex].nextIndex {

								nextIndexLog := leaderServerState.Log[serverLeaderStates[serverIndex].nextIndex-1]
								prevLogIndex := serverLeaderStates[serverIndex].nextIndex-2
								prevLogTerm := 0
								if prevLogIndex > 0 && prevLogIndex <= len(leaderServerState.Log) {
									prevLogTerm = leaderServerState.Log[prevLogIndex].Term
								}
								go sendAppendEntriesMessage(
									serverAppendEntriesCom,
									[]LogEntry{nextIndexLog},
									leaderServerState,
									prevLogIndex,
									prevLogTerm,
									)
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
	appendEntriesCom AppendEntriesCom,
	entries []LogEntry,
	leaderServerState * ServerState,
	prevLogIndex int,
	prevLogTerm int,
	) {

	appendEntriesCom.message <- AppendEntriesMessage {
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

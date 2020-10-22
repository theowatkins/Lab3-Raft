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
	state *ServerState,
	onWinChannel *chan bool,
	serverStateLock *sync.Mutex,
	appendEntriesCom *[8]AppendEntriesCom,
	clientCommunicationChannel *chan KeyValue,
	persister Persister,
	) {
	//TODO: What if received message from new leader before onWinChannel gets to hear about it?
	for {
		select {
		case <-* onWinChannel: // got enough votes
			serverStateLock.Lock()
			state.Role = LeaderRole
			serverStateLock.Unlock()

			// initialize leader state
			var leaderState [ClusterSize]LeaderState
			for i := 0; i < ClusterSize; i++ {
				// Note that logentries are indexed from 1
				leaderState[i] = LeaderState{len(state.Log) + 1, 0}
			}

			go runHeartbeatThread(state, appendEntriesCom) // Implements L1.
			go readAndDistributeClientRequests(state, &leaderState, appendEntriesCom, clientCommunicationChannel, persister)
		}
	}
}

func runHeartbeatThread(
	state *ServerState,
	appendEntriesCom *[ClusterSize]AppendEntriesCom) {
	for state.Role == LeaderRole {
		for _, leaderCommunicationChannel := range *appendEntriesCom {
			leaderCommunicationChannel.message <- AppendEntriesMessage{
				// leader's term
				state.CurrentTerm,

				// leader's ID
				state.ServerId,

				// index of last entry in log
				len(state.Log),

				// term of last entry in log
				state.CurrentTerm,

				// list of logentries to store
				// ** empty for heartbeat **
				[]LogEntry{},

				// leader's current commit index
				state.commitIndex}
		}
		time.Sleep(time.Duration(HeartBeatDelay) * time.Millisecond)
	}
}

//TODO: update commitindex on majority
//TODO: respond to client
func readAndDistributeClientRequests(
	state *ServerState,
	serverLeaderStates *[ClusterSize]LeaderState,
	appendEntriesCom *[ClusterSize]AppendEntriesCom,
	clientCommunicationChannel *chan KeyValue,
	persister Persister,
	) {

	censusedReachedChannel := make(chan bool)

	nodesWithReplicatedEntry := 0
	/* AppendEntriesRequest Handler
	 * Distributes a client request to all servers in the system.
	 */
	go func () {
		for state.Role == LeaderRole {
			select {
			case clientRequest := <-*clientCommunicationChannel:

				clientLogEntry := LogEntry{state.CurrentTerm, clientRequest}

				for serverIndex, leaderCommunicationChannel := range *appendEntriesCom {
					go sendAppendEntriesMessage(leaderCommunicationChannel, []LogEntry{clientLogEntry}, state, serverLeaderStates[serverIndex])
				}

				state.Log = append(state.Log, LogEntry{state.CurrentTerm, clientRequest})
				state.lastApplied++

				err := persister.Save(strconv.Itoa(len(state.Log)), clientLogEntry)
				for err != nil { //retry until no error.
					err = persister.Save(strconv.Itoa(len(state.Log)), clientLogEntry)
				}

				<- censusedReachedChannel
				state.commitIndex++
				nodesWithReplicatedEntry = 0 //clear count for next client requests
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
				for state.Role == LeaderRole {
					select {
					case r := <-serverAppendEntriesCom.response:
						// TODO: why does the paper say to respond with a term?!
						if r.success {
							// update matchIndex and nextIndex on successful appendEntry
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
							fmt.Println("Error processing AppendEntries request: ", r.message)
							// resend message with more logEntries on failure
							serverLeaderStates[serverIndex].nextIndex -= 1 //
							nextIndexLog := state.Log[serverLeaderStates[serverIndex].nextIndex]
							go sendAppendEntriesMessage(
								serverAppendEntriesCom,
								[]LogEntry{nextIndexLog},
								state,
								serverLeaderStates[serverIndex])
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
	state * ServerState,
	leaderState LeaderState,
	) {
	prevLogIndex := leaderState.nextIndex - 1
	prevLogTerm := -1
	if prevLogIndex >= 1 { //recall indexing starts at 1
		prevLogTerm = state.Log[prevLogIndex].Term
	}

	appendEntriesCom.message <- AppendEntriesMessage {
		// leader's term
		state.CurrentTerm,

		// leader's ID
		state.ServerId,

		// index of previous entry in log
		prevLogIndex,

		// term of previous entry in log
		prevLogTerm,

		// new LogEntries to store
		// from nextIndex to end of log
		entries,

		// leader's current commit index
		state.commitIndex}
}

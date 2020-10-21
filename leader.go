package main

import (
	"fmt"
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
	clientCommunicationChannel *chan KeyValue) {
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
			go readAndDistributeClientRequests(state, &leaderState, appendEntriesCom, clientCommunicationChannel)
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
	clientCommunicationChannel *chan KeyValue) {

	/* AppendEntriesRequest Handler
	 * Distributes a client request to all servers in the system.
	 */
	for state.Role == LeaderRole {
		select {
		case clientRequest := <-*clientCommunicationChannel:
			fmt.Println("received entry from client.")

			for serverIndex, leaderCommunicationChannel := range *appendEntriesCom {
				go sendAppendEntriesMessage(leaderCommunicationChannel, state, serverLeaderStates[serverIndex])
			}

			//TODO: Leader should not commit to state until signal that majority received log entry
			state.Log = append(state.Log, LogEntry{state.CurrentTerm, clientRequest})
			state.lastApplied++
		}
	}

	/* AppendEntriesResponse Handlers
	 * For each server a goroutine is created that continuously reads AppendEntries resopnses
	 */
	for serverIndex, leaderCommunicationChannel := range *appendEntriesCom {
		leaderCommunicationChannel := leaderCommunicationChannel
		serverIndex := serverIndex
		go func() {
			for state.Role == LeaderRole {
				select {
				case r := <-leaderCommunicationChannel.response:
					// TODO: why does the paper say to respond with a term?!
					if r.success {
						// update matchIndex and nextIndex on successful appendEntry
						serverLeaderStates[serverIndex].matchIndex = r.message.PrevLogIndex + len(r.message.Entries)
						serverLeaderStates[serverIndex].nextIndex = serverLeaderStates[serverIndex].matchIndex + 1
					} else {
						// resend message with more logEntries on failure
						serverLeaderStates[serverIndex].nextIndex -= 1
						go sendAppendEntriesMessage(leaderCommunicationChannel, state, serverLeaderStates[serverIndex])
					}
				default:
					// do nothing
				}
			}
		}()
	}
}

func sendAppendEntriesMessage(appendEntriesCom AppendEntriesCom, state *ServerState, leaderState LeaderState) {
	appendEntriesCom.message <- AppendEntriesMessage {
		// leader's term
		state.CurrentTerm,

		// leader's ID
		state.ServerId,

		// index of previous entry in log
		leaderState.nextIndex - 1,

		// term of previous entry in log
		state.Log[leaderState.nextIndex - 1].Term,

		// new LogEntries to store
		// from nextIndex to end of log
		state.Log[leaderState.nextIndex - 1:],

		// leader's current commit index
		state.commitIndex}
}

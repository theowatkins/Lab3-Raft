package main

import (
	"sync"
)
import "time"

const HeartBeatDelay = 500
const ClusterSize = 8
const ElectionTimeOut = 5 * 1000 // in milliseconds
var debug = false

type Vote struct {
	Term int
	VoteFor int
	Responses chan bool
}

const UndefinedIndex = -1

func initCluster(clientCommunicationChannel chan KeyValue, persister Persister) {

	var voteChannels [ClusterSize]chan Vote
	var appendEntriesCom [ClusterSize]AppendEntriesCom

	previousLogEntries := initializeServerStateFromPersister(persister)

	// Spawn 8 nodes (all followers to start)
	for i := 0; i < ClusterSize; i++ {
		// initialize state as followers
		state := ServerState{i, 0, -1, previousLogEntries, FollowerRole, UndefinedIndex,UndefinedIndex}

		voteChannels[i] = make(chan Vote)
		appendEntriesCom[i] = AppendEntriesCom{make(chan AppendEntriesMessage), make(chan AppendEntriesResponse)}

		go startServer(&state, &voteChannels, &appendEntriesCom, clientCommunicationChannel, persister)
	}
}

/* Creates a server in the cluster. Structured via https://pdos.csail.mit.edu/6.824/labs/raft-structure.txt
 *
 */
func startServer(
	state *ServerState,
	voteChannels *[ClusterSize]chan Vote,
	appendEntriesCom *[ClusterSize]AppendEntriesCom,
	clientCommunicationChannel chan KeyValue,
	persister Persister,
	) {

	isElection := true
	electionThreadSleepTime := time.Millisecond * 1000
	timeSinceLastUpdate := time.Now() //update includes election or message from leader
	serverStateLock := new(sync.Mutex)
	onWinChannel := make(chan bool)

	go runElectionTimeoutThread(&timeSinceLastUpdate, &isElection, state, voteChannels, &onWinChannel, electionThreadSleepTime)
	go startLeaderListener(appendEntriesCom, state, &timeSinceLastUpdate, &isElection, serverStateLock)
	go onWinChannelListener(state, &onWinChannel, serverStateLock, appendEntriesCom, &clientCommunicationChannel, persister) //in leader.go
}

/* Election Timer: Checks if timeout is surpassed and starts election. Timeout is reached when:
 * 1. no message from leader or
 * 2. when election took too long (e.g. due to tie / no leader elected)
 */
func runElectionTimeoutThread(
	timeSinceLastUpdate * time.Time,
	isElection * bool,
	state * ServerState,
	voteChannels *[8]chan Vote,
	onWinChannel * chan bool,
	electionThreadSleepTime time.Duration,
) {
	for {
		timeElapsed := time.Now().Sub(*timeSinceLastUpdate)
		if timeElapsed.Milliseconds() > ElectionTimeOut { //implements C4.
			*isElection = true
		}

		if *isElection {
			*timeSinceLastUpdate = time.Now()
			go elect(state, voteChannels, *onWinChannel)
		}

		time.Sleep(electionThreadSleepTime)
	}
}

/* Handles messages from leader. Duties include:
* - ignore anything with stale term
 * - update timeSinceLastUpdate
 * - process new log entries + heartbeats (empty logs)
*/
func startLeaderListener(
	appendEntriesCom * [8]AppendEntriesCom,
	state * ServerState,
	timeSinceLastUpdate * time.Time,
	isElection * bool,
	serverStateLock * sync.Mutex,
	) {
	for {
		select {
		case appendEntryRequest := <-appendEntriesCom[state.ServerId].message:
			if appendEntryRequest.Term >= state.CurrentTerm {
				*timeSinceLastUpdate = time.Now()
				state.CurrentTerm = appendEntryRequest.Term
				if *isElection { //received message from leader during election,
					onElectionEndHandler(isElection, serverStateLock, state)
				}

				printMessageFromLeader(state.ServerId, appendEntryRequest)
				if state.Role != LeaderRole { //processed separately before all others
					processAppendEntryRequest(appendEntryRequest, state, appendEntriesCom)
				}
			}
		}
	}
}

func onElectionEndHandler(isElection * bool, serverStateLock *sync.Mutex, state *ServerState) {
	*isElection = false
	serverStateLock.Lock()
	if state.Role != LeaderRole {
		state.Role = FollowerRole // for candidates that lost the election
	}
	serverStateLock.Unlock()
}

func processAppendEntryRequest(appendEntryRequest AppendEntriesMessage, state *ServerState, appendEntriesCom *[8]AppendEntriesCom) {
	if len(appendEntryRequest.Entries) > 0 {
		if appendEntryRequest.PrevLogIndex == 0 {
			for _, entry := range appendEntryRequest.Entries {
				state.Log = append(state.Log, entry)
			}
			appendEntriesCom[state.ServerId].response <- AppendEntriesResponse{state.CurrentTerm, true, appendEntryRequest}
		} else if appendEntryRequest.PrevLogIndex > len(state.Log) ||
			state.Log[appendEntryRequest.PrevLogIndex-1].Term != appendEntryRequest.PrevLogTerm {
			// respond to leader, append failed (need more entries)
			appendEntriesCom[state.ServerId].response <- AppendEntriesResponse{state.CurrentTerm, false, appendEntryRequest}
		} else {
			// update log
			for i, entry := range appendEntryRequest.Entries {
				state.Log = append(state.Log[:(appendEntryRequest.PrevLogIndex-1)+i], entry)
			}

			// respond to leader, append succeeded
			appendEntriesCom[state.ServerId].response <- AppendEntriesResponse{state.CurrentTerm, true, appendEntryRequest}
		}
	}
}

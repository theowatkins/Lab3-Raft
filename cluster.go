package main

import (
	"sync"
	"fmt"
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
	LastLogIndex int
	LastLogTerm int
}

const UndefinedIndex = -1

func initCluster(clientCommunicationChannel chan KeyValue, persister Persister, applyChannel ApplyChannel) {

	var voteChannels [ClusterSize]chan Vote
	var appendEntriesCom [ClusterSize]AppendEntriesCom

	// Spawn 8 nodes (all followers to start)
	for serverIndex := 0; serverIndex < ClusterSize; serverIndex++ {
		voteChannels[serverIndex] = make(chan Vote)
		appendEntriesCom[serverIndex] = AppendEntriesCom{make(chan AppendEntriesMessage), make(chan AppendEntriesResponse)}
	}

	networkIdentifiers := NetworkIdentifiers{}
	networkIdentifiers.voteChannels = &voteChannels
	networkIdentifiers.appendEntriesCom = &appendEntriesCom
	networkIdentifiers.clientCommunicationChannel = clientCommunicationChannel

	for serverIndex := 0; serverIndex < ClusterSize; serverIndex++ {
		MakeRaft(networkIdentifiers, serverIndex, persister, applyChannel)
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
	channel ApplyChannel,
	) Raft {

	isElection := true
	electionThreadSleepTime := time.Millisecond * 1000
	timeSinceLastUpdate := time.Now() //update includes election or message from leader
	serverStateLock := new(sync.Mutex)
	onWinChannel := make(chan bool)

	go runElectionTimeoutThread(&timeSinceLastUpdate, &isElection, state, voteChannels, &onWinChannel, electionThreadSleepTime)
	go startLeaderListener(appendEntriesCom, state, &timeSinceLastUpdate, &isElection, serverStateLock) //implements F1.
	go onWinChannelListener(state, &onWinChannel, serverStateLock, appendEntriesCom, &clientCommunicationChannel, persister, channel) //in leader.go

	//creates raft object with closure
	raft := Raft{}
	raft.Start = func (logEntry LogEntry) (int, int, bool){ //implements
		go func () { //non blocking sent through client (leader may not be choosen yet).
			clientCommunicationChannel <- logEntry.Content
		}()
		return len(state.Log), state.CurrentTerm, state.Role == LeaderRole
	}

	raft.GetState = func ()(int, bool) {
		return state.CurrentTerm, state.Role == LeaderRole
	}
	return raft
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
				state.CurrentTerm = appendEntryRequest.Term //implements AS2.
				if *isElection { //received message from leader during election,
					onElectionEndHandler(isElection, serverStateLock, state)
				}
				
				printMessageFromLeader(state.ServerId, appendEntryRequest)
				if state.Role != LeaderRole && len(appendEntryRequest.Entries) != 0 { //implements C3
					processAppendEntryRequest(appendEntryRequest, state, appendEntriesCom)
					fmt.Println("Server ", state.ServerId, "'s current log: ", state.Log)
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
	onFail := func () {
		appendEntriesCom[state.ServerId].response <- AppendEntriesResponse{state.CurrentTerm, false, appendEntryRequest}
	}

	onSuccess := func() {
		appendEntriesCom[state.ServerId].response <- AppendEntriesResponse{state.CurrentTerm, true, appendEntryRequest}
	}

	if appendEntryRequest.LeaderCommit > state.commitIndex { //implements AE5.
		state.commitIndex = Min(appendEntryRequest.LeaderCommit, len(state.Log))
	}

	var termCompare int
	if appendEntryRequest.PrevLogIndex == 0 {
		termCompare = -1
	} else {
		termCompare = state.Log[appendEntryRequest.PrevLogIndex - 1].Term
	}

	if state.CurrentTerm < appendEntryRequest.Term { //implements AE1.
		onFail()
		return
	} else if appendEntryRequest.PrevLogIndex <= len(state.Log) && appendEntryRequest.PrevLogTerm == termCompare { //implements AE3.
		state.Log = append(state.Log[:appendEntryRequest.PrevLogIndex], appendEntryRequest.Entries...) //not shifting index because slice ignores upper bound
		onSuccess()
		return
	} else { //implements AE2.
		// respond to leader, append failed (need more entries)
		onFail()
		return
	}
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

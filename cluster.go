package main

import (
	"fmt"
	"math/rand"
	"sync"
)
import "time"

const ClusterSize = 8
const ElectionTimeOut = 2 * 1000 // in milliseconds

type Vote struct {
	Term int
	VoteFor int
	Responses chan bool
}

func initCluster(done chan bool) {

	var voteChannels [ClusterSize]chan Vote
	var leaderCommunicationChannel [ClusterSize] chan LogEntry

	// Spawn 8 nodes (all followers to start)
	for i := 0; i < ClusterSize; i++ {
		// initialize state as followers
		state := ServerState{i, 0, -1, []LogEntry{}, FollowerRole}

		voteChannels[i] = make(chan Vote)
		leaderCommunicationChannel[i] = make(chan LogEntry)

		go startServer(state, &voteChannels, &leaderCommunicationChannel, done)
	}
}

/* Creates a server in the cluster. Structured via https://pdos.csail.mit.edu/6.824/labs/raft-structure.txt
 *
 */
func startServer(
	state ServerState,
	voteChannels *[ClusterSize]chan Vote,
	leaderCommunicationChannels *[ClusterSize]chan LogEntry,
	done chan bool,
) {
	isElection := false
	electionThreadSleepTime := time.Millisecond * 50
	timeSinceLastUpdate := time.Now() //update includes election or message from leader
	serverStateLock := new(sync.Mutex)

	/* Election Timer: Checks if timeout is surpassed and starts election. Timeout is reached when:
	 * 1. no message from leader or
	 * 2. when election took too long (e.g. due to tie / no leader elected)
	 */
	go func () {
		for {
			timeElapsed := time.Now().Sub(timeSinceLastUpdate)
			if timeElapsed.Milliseconds() > ElectionTimeOut {
				isElection = true
				timeSinceLastUpdate = time.Now()
				go elect(&state, voteChannels, leaderCommunicationChannels)
			}
			time.Sleep(electionThreadSleepTime)
		}
	}()

	// receive messages from leader
	go func () {
		for newLogEntry := range leaderCommunicationChannels[state.ServerId] {
			if isElection { //received message from leader during election,
				serverStateLock.Lock()
				state.Role = FollowerRole // for candidates that lost the election
				serverStateLock.Unlock()
				isElection = false
			}
			timeSinceLastUpdate = time.Now()
			fmt.Print("New log entry received from leader:", newLogEntry, "\n")
			//process log entry here
		}
		done <- true
	}()
}

/* Begins an election and handles the following events:
 * 1. Election timeout reaches threshold -> Become candidate
 * 2. External Request to vote -> Give vote, ignore if voted already
 */
func elect(
	state * ServerState,
	voteChannels *[ClusterSize]chan Vote,
	leaderCommunicationChannels *[ClusterSize]chan LogEntry,
	) {
	timeUntilElectionStart := rand.Intn(150) + 150
	electionStartTimer := time.NewTimer(time.Duration(timeUntilElectionStart) * time.Millisecond)
	serverStateLock := new(sync.Mutex)

	select {
	case <-electionStartTimer.C: // candidate
		fmt.Println("Server ", state.ServerId, " is a candidate")

		// lock while transitioning to candidate
		serverStateLock.Lock()
		state.CurrentTerm++
		state.Role = CandidateRole
		state.VotedFor = state.ServerId // vote for self
		serverStateLock.Unlock()
		
		//count votes
		winnerChannel := make(chan bool)
		go requestVotes(state, voteChannels, winnerChannel)
		
		
		select {
		case <-winnerChannel: // got enough votes
			for serverIndex, leaderCommunicationChannel := range *leaderCommunicationChannels {
				serverStateLock.Lock()
				state.Role = LeaderRole
				serverStateLock.Unlock()
				leaderCommunicationChannel <- LogEntry{serverIndex, state.CurrentTerm, KeyValue{"", ""}}
			}
		}
	// CurrentTerm is used as a flag to identify if a server has voted.
	case voteRequest := <-(*voteChannels)[state.ServerId]: // follower (being asked to vote)
		serverStateLock.Lock()
		if voteRequest.Term > state.CurrentTerm { // I haven't voted yet (noted by stale term)
			state.CurrentTerm = voteRequest.Term
			state.VotedFor = voteRequest.VoteFor
			serverStateLock.Unlock()

			voteRequest.Responses <- true
			fmt.Println("Server ", state.ServerId, " voted for ", state.VotedFor)
		} else { // I already voted
			voteRequest.Responses <- false
		}
	}
}

func requestVotes(state * ServerState, voteChannels *[ClusterSize]chan Vote, winnerChannel chan bool) {
	// send vote requests to other servers
	responses := make(chan bool)
	for i, c := range *voteChannels {
		if i != state.ServerId {
			c <- Vote{state.CurrentTerm, state.ServerId, responses}
		}
	}
	
	// count votes
	votes := 0
	for j := 0; j < (ClusterSize - 1); j++ {
		r := <-responses
		if r {
			votes++
		}
	}

	if votes >= ClusterSize/2 { // won election
		fmt.Println("Server ", state.ServerId, " is the leader!")
		winnerChannel <- true
	} else {
		fmt.Println("Server ", state.ServerId, " lost election.")
	}
}

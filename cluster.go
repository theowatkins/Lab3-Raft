package main

import "fmt"
import "time"
import "math/rand"

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
		state := ServerState{i, 0, -1, []LogEntry{}}

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
	electionThreadSleepTime := time.Millisecond * 50
	timeSinceLastUpdate := time.Now() //update includes election or message from leader

	/* Election Timer: Checks if timeout is surpassed and starts election. Timeout is reached when:
	 * 1. no message from leader or
	 * 2. when election took too long (e.g. due to tie / no leader elected)
	 */
	go func () {
		for {
			timeElapsed := time.Now().Sub(timeSinceLastUpdate)
			if timeElapsed.Milliseconds() > ElectionTimeOut {
				timeSinceLastUpdate = time.Now()
				go elect(state, voteChannels, leaderCommunicationChannels)
			}
			time.Sleep(electionThreadSleepTime)
		}
	}()

	// receive messages from leader
	go func () {
		for newLogEntry := range leaderCommunicationChannels[state.ServerId] {
			timeSinceLastUpdate = time.Now()
			fmt.Print("New log entry received from leader:", newLogEntry, "\n")
			//process log entry here
		}
		done <- true
	}()
}

/* Begins an election.
 *
 */
func elect(
	state ServerState,
	voteChannels *[ClusterSize]chan Vote,
	leaderCommunicationChannels *[ClusterSize]chan LogEntry,
	) {
	timeUntilElectionStart := rand.Intn(150) + 150
	electionStartTimer := time.NewTimer(time.Duration(timeUntilElectionStart) * time.Millisecond)

	select {
	case <-electionStartTimer.C: // candidate
		fmt.Println("Server ", state.ServerId, " is a candidate")
		// start election
		state.CurrentTerm++

		// vote for self
		state.VotedFor = state.ServerId
		
		//count votes
		winnerChannel := make(chan bool)
		go requestVotes(state, voteChannels, winnerChannel)
		
		select {
		case <-winnerChannel: // got enough votes
			for serverIndex, leaderCommunicationChannel := range leaderCommunicationChannels {
				leaderCommunicationChannel := leaderCommunicationChannel
				go func () {
					leaderCommunicationChannel <- LogEntry{serverIndex, state.CurrentTerm, KeyValue{"assert", "dominance"}}
				}()
			}
		}
		
	case v := <-(*voteChannels)[state.ServerId]: // follower
		if v.Term > state.CurrentTerm { // I haven't voted yet
			state.CurrentTerm = v.Term
			state.VotedFor = v.VoteFor
			v.Responses <- true
			fmt.Println("Server ", state.ServerId, " voted for ", state.VotedFor)
		} else { // I already voted
			v.Responses <- false
			fmt.Println("Server ", state.ServerId, " didn't vote for ", v.VoteFor, " because it already voted in term ", v.Term)
		}
	}
}

func requestVotes(state ServerState, voteChannels *[ClusterSize]chan Vote, winnerChannel chan bool) {
	// send vote requests to other servers
	responses := make(chan bool)
	for i, c := range (*voteChannels) {
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

	fmt.Println("Server ", state.ServerId, " received ", votes, " votes")
	if votes >= ClusterSize/2 { // won election
		fmt.Println("Server ", state.ServerId, " is the leader!")
		winnerChannel <- true
	}
}

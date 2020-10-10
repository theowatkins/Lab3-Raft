package main

import (
	"fmt"
	"math/rand"
	"sync"
)
import "time"

const HeartBeatDelay = 1000
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
	onWinChannel := make(chan bool)

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
				go elect(&state, voteChannels, onWinChannel)
			}
			time.Sleep(electionThreadSleepTime)
		}
	}()

	/* Handles messages from leader. Duties include:
	* - ignore anything with stale term
	 * - update timeSinceLastUpdate
	 * - process new log entries + heartbeats (empty logs)
	 */
	go func () {
		for newLogEntry := range leaderCommunicationChannels[state.ServerId] {
			 if newLogEntry.Term >= state.CurrentTerm {
				timeSinceLastUpdate = time.Now()

				if isElection { //received message from leader during election,
					isElection = false
					serverStateLock.Lock()
					if state.Role != LeaderRole {
						state.Role = FollowerRole // for candidates that lost the election
					}
					serverStateLock.Unlock()
				}

				printMessageFromLeader(state.ServerId, newLogEntry)
				//process log entry here
			}

		}
		done <- true
	}()

	/* On Win Handler
	 *
	 */
	go func(){
		select {
		case <-onWinChannel: // got enough votes
			serverStateLock.Lock()
			state.Role = LeaderRole
			serverStateLock.Unlock()
			for state.Role == LeaderRole {
				for serverIndex, leaderCommunicationChannel := range *leaderCommunicationChannels {
					leaderCommunicationChannel <- LogEntry{serverIndex, state.CurrentTerm, KeyValue{"", ""}}
				}
				fmt.Println() //breaks up prints into chunks for each beat.
				time.Sleep(HeartBeatDelay * time.Millisecond)
			}
		}
	}()
}

func printMessageFromLeader(id int, logEntry LogEntry){
	if logEntry.Content.Key == "" &&
		logEntry.Content.Value == "" {
		fmt.Println(id, " received heartbeat from leader.")
	} else {
		fmt.Println(id, " received new entry: ", logEntry)
	}

}


/* Begins an election and handles the following events:
 * 1. Election timeout reaches threshold -> Become candidate
 * 2. External Request to vote -> Give vote, ignore if voted already
 */
func elect(
	state * ServerState,
	voteChannels *[ClusterSize]chan Vote,
	onWinChannel chan bool,
	) {
	timeUntilElectionStart := rand.Intn(150) + 150
	electionStartTimer := time.NewTimer(time.Duration(timeUntilElectionStart) * time.Millisecond)
	serverStateLock := new(sync.Mutex)

	select {
	case <-electionStartTimer.C: // candidate
		fmt.Println("Server ", state.ServerId, " is a candidate for term: ", state.CurrentTerm + 1)

		// lock while transitioning to candidate
		serverStateLock.Lock()
		state.CurrentTerm++
		state.Role = CandidateRole
		state.VotedFor = state.ServerId // vote for self
		serverStateLock.Unlock()
		
		//count votes
		go requestVotes(state, voteChannels, onWinChannel)


	/* Response handler for vote requests.
	 * Note, CurrentTerm is used as a flag to identify if a server has voted.
	 * If vote request contains a future term, then vote is confirmed and CurrentTerm updated to reject any
	 * other candidate running in the same term.
	 */
	case voteRequest := <-(*voteChannels)[state.ServerId]: //
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

func requestVotes(state * ServerState, voteChannels *[ClusterSize]chan Vote, onWinChannel chan bool) {
	// send vote requests to other servers
	//TODO: What if one of the servers is down -- then this will hang forever
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
		fmt.Print("Server ", state.ServerId, " is the leader!\n\n")
		onWinChannel <- true
	} else {
		fmt.Println("Server ", state.ServerId, " lost the election.")
	}
}

package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

/* Begins an election and handles the following events:
 * 1. Election timeout reaches threshold -> Become candidate
 * 2. External Request to vote -> Give vote, ignore if voted already
 */
func elect(
	state *ServerState,
	voteChannels *[ClusterSize]chan Vote,
	onWinChannel chan bool,
) {
	timeUntilElectionStart := rand.Intn(150) + 150
	electionStartTimer := time.NewTimer(time.Duration(timeUntilElectionStart) * time.Millisecond)
	serverStateLock := new(sync.Mutex)

	select {
	case <-electionStartTimer.C: // candidate
		// implements C1
		serverStateLock.Lock()
		state.Role = CandidateRole
		state.CurrentTerm++
		state.VotedFor = state.ServerId // vote for self
		serverStateLock.Unlock()

		//count votes
		go requestVotes(state, voteChannels, onWinChannel)

		time.Sleep(time.Duration(timeUntilElectionStart) * time.Millisecond) //reset election timer

		//continue elections until you win or turn into a follower when leader sends heartbeats
		for state.Role == CandidateRole {
			serverStateLock.Lock()
			state.CurrentTerm++
			go requestVotes(state, voteChannels, onWinChannel)
			serverStateLock.Unlock()
		}

	/* Response handler for vote requests.
	 * Note, CurrentTerm is used as a flag to identify if a server has voted.
	 * If vote request contains a future term, then vote is confirmed and CurrentTerm updated to reject any
	 * other candidate running in the same term.
	 */
	case voteRequest := <-(*voteChannels)[state.ServerId]: //implements F1.
		serverStateLock.Lock()
		if state.VotedFor < 0 || state.VotedFor == voteRequest.VoteFor {
			// if votedFor is null or candidateId
			var lastLogTerm int
			lastLogTerm = 0
			lastLogIndex := len(state.Log)
			if lastLogIndex > 0 {
				lastLogTerm = state.Log[voteRequest.LastLogIndex-1].Term
			}

			if voteRequest.Term > state.CurrentTerm && //implements RV2.
				voteRequest.LastLogIndex >= lastLogIndex &&
				voteRequest.LastLogTerm == lastLogTerm{
				// candidates log is at least as up-to-date as receiver's log
				state.VotedFor = voteRequest.VoteFor
				voteRequest.Responses <- true
			} else { //implements RV1.
				voteRequest.Responses <- false
			}
		}
	}
}

func requestVotes(state *ServerState, voteChannels *[ClusterSize]chan Vote, onWinChannel chan bool) {
	// send vote requests to other servers
	//TODO: What if one of the servers is down -- then this will hang forever
	responses := make(chan bool)
	var candidateLastLogTerm int
	candidateLastLogTerm = 0
	candidateLastLogIndex := len(state.Log)
	if candidateLastLogIndex > 0 {
		candidateLastLogTerm = state.Log[candidateLastLogIndex-1].Term
	}
	
	for i, c := range *voteChannels {
		if i != state.ServerId {
			c <- Vote{state.CurrentTerm, state.ServerId, responses, candidateLastLogIndex, candidateLastLogTerm}
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

	if votes >= ClusterSize/2 { // implements c2.
		fmt.Println("Server ", state.ServerId, " is the leader!\n\n")
		onWinChannel <- true
	} else {
		fmt.Println("Server ", state.ServerId, " lost the election.")
	}
}

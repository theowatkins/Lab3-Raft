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
	s1 := rand.NewSource(time.Now().UnixNano())
    r1 := rand.New(s1)
	timeUntilElectionStart := r1.Intn(150) + 150
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
		
		if voteRequest.Term > state.CurrentTerm {//implements AS2
			staleTerm(state, voteRequest.Term)
		}
		
		if voteRequest.Term < state.CurrentTerm {// implements RV1
			voteRequest.Responses <- VoteResponse{false, state.CurrentTerm}
		} else if state.VotedFor < 0 || state.VotedFor == voteRequest.VoteFor { //implements RV2.1
			// if votedFor is null or candidateId
			var lastLogTerm int
			lastLogTerm = 0
			lastLogIndex := len(state.Log)
			if voteRequest.LastLogIndex > 0 && voteRequest.LastLogIndex <= lastLogIndex {
				lastLogTerm = state.Log[voteRequest.LastLogIndex-1].Term
			}

			if voteRequest.LastLogIndex >= lastLogIndex &&
				voteRequest.LastLogTerm == lastLogTerm {//implements RV2.2

				// candidates log is at least as up-to-date as receiver's log
				state.VotedFor = voteRequest.VoteFor
				voteRequest.Responses <- VoteResponse{true, state.CurrentTerm}
			} else {
				voteRequest.Responses <- VoteResponse{false, state.CurrentTerm}
			}
		} else {
			voteRequest.Responses <- VoteResponse{false, state.CurrentTerm}
		}
	}
}

func requestVotes(state *ServerState, voteChannels *[ClusterSize]chan Vote, onWinChannel chan bool) {
	// send vote requests to other servers
	responses := make(chan VoteResponse)
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
	go func() {
		votes := 0
		for j := 0; j < (ClusterSize - 1); j++ {
			r := <-responses
			if r.Term > state.CurrentTerm { // implements AS2
				staleTerm(state, r.Term)
				break
			}
			if r.GotVote  {
				votes++
				if votes >= ClusterSize/2 { // implements C2
					fmt.Println("Server ", state.ServerId, " is the leader!")
					onWinChannel <- true
					break
				} 
			}
		}
	}()
}

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
		} else { //implements RV1.
			voteRequest.Responses <- false
		}
	}
}

func requestVotes(state *ServerState, voteChannels *[ClusterSize]chan Vote, onWinChannel chan bool) {
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

	if votes >= ClusterSize/2 { // implements c2.
		fmt.Print("Server ", state.ServerId, " is the leader!\n\n")
		onWinChannel <- true
	} else {
		fmt.Println("Server ", state.ServerId, " lost the election.")
	}
}

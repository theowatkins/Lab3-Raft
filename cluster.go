// cluster.go

package main

import "fmt"
import "time"
import "math/rand"

const ClusterSize = 8
const ElectionTimeOut = 1000

type Vote struct {
	Term int
	VoteFor int
	Responses chan bool
}


func initCluster(done chan bool) {

	var voteChannels [ClusterSize]chan Vote

	// Spawn 8 nodes (all followers to start)
	for i := 0; i < ClusterSize; i++ {
		// initialize state as followers
		state := ServerState{i, 0, -1, []LogEntry{}}

		voteChannels[i] = make(chan Vote)

		go startServer(state, &voteChannels, done)
	}
}


func startServer(state ServerState, voteChannels *[ClusterSize]chan Vote, done chan bool) {
	//start election timer
	electionTime := time.NewTimer(time.Duration(ElectionTimeOut) * time.Millisecond)

	elect(state, voteChannels, electionTime)
	
	done <- true
}


func elect(state ServerState, voteChannels *[ClusterSize]chan Vote, electionTime *time.Timer) {
	startTimeOut := rand.Intn(150) + 150

	startTime := time.NewTimer(time.Duration(startTimeOut) * time.Millisecond)

	select {
	case <-startTime.C: // candidate
		fmt.Println("Server ", state.ServerId, " is a candidate")
		// start election
		state.CurrentTerm++

		// vote for self
		state.VotedFor = state.ServerId
		
		// reset election timer
		electionTime.Reset(time.Duration(ElectionTimeOut) * time.Millisecond)
		
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
		} else { // lost election
			fmt.Println("Server ", state.ServerId, " didn't get enough votes :(")
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
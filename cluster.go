package main

import (
	"fmt"
	"math/rand"
	"sync"
)
import "time"

const HeartBeatDelay = 500
const ClusterSize = 8
const ElectionTimeOut = 5 * 1000 // in milliseconds

type Vote struct {
	Term int
	VoteFor int
	Responses chan bool
}

func initCluster(clientCommunicationChannel chan KeyValue, persister Persister) {

	var voteChannels [ClusterSize]chan Vote
	var leaderCommunicationChannel [ClusterSize]LeaderCom

	previousLogEntries := initializeServerStateFromPersister(persister)

	// Spawn 8 nodes (all followers to start)
	for i := 0; i < ClusterSize; i++ {
		// initialize state as followers
		state := ServerState{i, 0, -1, previousLogEntries, FollowerRole, 0,0}

		voteChannels[i] = make(chan Vote)
		leaderCommunicationChannel[i] = LeaderCom{make(chan AppendEntriesMessage), make(chan AppendEntriesResponse)}

		go startServer(&state, &voteChannels, &leaderCommunicationChannel, clientCommunicationChannel)
	}
}

/* Creates a server in the cluster. Structured via https://pdos.csail.mit.edu/6.824/labs/raft-structure.txt
 *
 */
func startServer(
	state *ServerState,
	voteChannels *[ClusterSize]chan Vote,
	leaderCommunicationChannels *[ClusterSize]LeaderCom,
	clientCommunicationChannel chan KeyValue) {

	isElection := false
	electionThreadSleepTime := time.Millisecond * 1000
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
				go elect(state, voteChannels, onWinChannel)
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
		for {
			select {
			case appendEntry := <-leaderCommunicationChannels[state.ServerId].message:
				if appendEntry.Term >= state.CurrentTerm {
					timeSinceLastUpdate = time.Now()
					state.CurrentTerm = appendEntry.Term

					if isElection { //received message from leader during election,
						isElection = false
						serverStateLock.Lock()
						if state.Role != LeaderRole {
							state.Role = FollowerRole // for candidates that lost the election
						}
						serverStateLock.Unlock()
					}

					printMessageFromLeader(state.ServerId, appendEntry)

					if len(appendEntry.Entries) > 0 {
						if appendEntry.PrevLogIndex < len(state.Log) && 
						  state.Log[appendEntry.PrevLogIndex].Term == appendEntry.PrevLogTerm {

							// update log
							for i, entry := range appendEntry.Entries{
								state.Log = append(state.Log[:appendEntry.PrevLogIndex + i], entry)
							}

							// respond to leader, append succeeded
							leaderCommunicationChannels[state.ServerId].response <-AppendEntriesResponse{state.CurrentTerm, true, appendEntry}
						} else {
							// respond to leader, apppend failed (need more entries)
							leaderCommunicationChannels[state.ServerId].response <-AppendEntriesResponse{state.CurrentTerm, false, appendEntry}
						}
					} else {
						// do nothing for heartbeats
					}
				}
			}
		}
	}()

	/* On Win Handler. Responsibilities includes
	 * - updating role of server to leader
	 * - running heartbeat thread
	 * - managing client requests (in progress)
	 */
	go func(){
		select {
		case <-onWinChannel: // got enough votes
			serverStateLock.Lock()
			state.Role = LeaderRole
			serverStateLock.Unlock()

			// initialize leader state
			var leaderState [ClusterSize]LeaderState
			for i:=0; i < ClusterSize; i++ {
				leaderState[i] = LeaderState{len(state.Log), 0}
			}

			go runHeartbeatThread(state, leaderCommunicationChannels)
			go readAndDistributeClientRequests(state, &leaderState, leaderCommunicationChannels, clientCommunicationChannel)
		}
	}()
}

func runHeartbeatThread(
	state * ServerState, 
	leaderCommunicationChannels *[ClusterSize]LeaderCom) {
	for state.Role == LeaderRole {
		for _, leaderCommunicationChannel := range *leaderCommunicationChannels {
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
		fmt.Println("sent") //breaks up prints into chunks for each beat.
		time.Sleep(time.Duration(HeartBeatDelay) * time.Millisecond)
	}
}

//TODO: update commitindex on majority
//TODO: respond to client
func readAndDistributeClientRequests(
	state * ServerState, 
	leaderState *[ClusterSize]LeaderState,
	leaderCommunicationChannels *[ClusterSize]LeaderCom,
	clientCommunicationChannel chan KeyValue) {

	for state.Role == LeaderRole {
		select {
		case clientRequest := <-clientCommunicationChannel:
			// Append to log
			state.Log = append(state.Log, LogEntry{state.CurrentTerm, clientRequest})
			state.lastApplied++

			for serverIndex, leaderCommunicationChannel := range *leaderCommunicationChannels {
				go sendAppend(leaderCommunicationChannel, state, leaderState[serverIndex])
			}
		default:
			// check response channels
			for serverIndex, leaderCommunicationChannel := range *leaderCommunicationChannels {
				select {
				case r := <- leaderCommunicationChannel.response:
					// TODO: why does the paper say to respond with a term?!
					if r.success {
						// update matchIndex and nextIndex on successful appendEntry
						leaderState[serverIndex].matchIndex = r.message.PrevLogIndex + len(r.message.Entries)
						leaderState[serverIndex].nextIndex = leaderState[serverIndex].matchIndex + 1
					} else {
						// resend message with more logEntries on failure
						leaderState[serverIndex].nextIndex -= 1
						go sendAppend(leaderCommunicationChannel, state, leaderState[serverIndex])
					}
				default:
					// do nothing
				}
			}
		} 
	}
}

func sendAppend(leaderCommunicationChannel LeaderCom, state *ServerState, leaderState LeaderState) {
	leaderCommunicationChannel.message <- AppendEntriesMessage{
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
		state.Log[leaderState.nextIndex:],

		// leader's current commit index
		state.commitIndex}
}

func printMessageFromLeader(id int, append AppendEntriesMessage){
	if len(append.Entries) > 0 {
		fmt.Println("Server ", id, " received new entry from ", append.LeaderId, ": ")
		printMessage(append)
	} else {
		fmt.Println(id, " received heartbeat from ", append.LeaderId)
	}
}

func printMessage(append AppendEntriesMessage) {
	fmt.Println("\tEntries to append: ", append.Entries)
	fmt.Println("\tPrevious index: ", append.PrevLogIndex)
	fmt.Println("\tPrevious term: ", append.PrevLogTerm)
	fmt.Println("\tLeader term: ", append.Term)
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

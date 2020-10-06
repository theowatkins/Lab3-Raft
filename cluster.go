// cluster.go

const ClusterSize = 8

// Spawn 8 nodes (all followers to start)
func initCluster() {

	// We gotta figure out how we want to use channels
	// rn here's what I'm thinkin:
	//     1. Election channel (int of ServerId to vote for)
	//         - might need an ack channel so we can check if a majority voted
	//           for a node and we can officially mark it as the leader
	//         - might need map of channels (one per follower)
	//     2. Append to log channel (LogEntry sent from leader to followers)
	//         - might need an ack channel so we can check if a majority acked
	//           the value and we should mark it as committed
	//         - might need map of channels (one per follower)

	for i := 0; i < ClusterSize; i++ {
		// Maybe we make a couple channel maps here and pass them to each node 
		// so that if it becomes the leader it can communicate with each
		// follower separately.  Just a thought, we've got a lot to discuss
		go func() {
			state := ServerState{i, -1, []LogEntry}

			// random timeout

			// wait for vote requests

			// if request not recieved, requestVote()


		}
	}
}

// Send a vote request and vote for yourself
func requestVote() {
    
}


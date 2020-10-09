// Lab3 - Raft

package main

import "fmt"

func main() {
	// 1. Make client to cluster channel (KeyValue to store on system)
	//    - might need a reject channel for when the cluster fails to 
	//      store the pair (meaning it does an election and starts a 
	//      new term) so the client needs to resend it

	// 2. Spawn cluster
	done := make(chan bool)
	initCluster(done)

	for i := 0; i < ClusterSize; i++ {
		<-done
	}

	fmt.Println("all done")

	// 3. Send Key Value pairs to cluster
}
package main

import (
	"fmt"
	"bufio"
	"os"
	"strings"
)

func main() {
	// 1. Make client to cluster channel (KeyValue to store on system)
	//    - might need a reject channel for when the cluster fails to 
	//      store the pair (meaning it does an election and starts a 
	//      new term) so the client needs to resend it

	// 2. Spawn cluster
	clientCommunicationChannel := make(chan KeyValue)
	fmt.Print("Creating cluster...\n")
	initCluster(clientCommunicationChannel, TestPersister{})

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter Key, Value pair: ")
		text, _ := reader.ReadString('\n')
		pair := strings.Split(text, ",")
		clientCommunicationChannel <-KeyValue{pair[0], pair[1]}
	}
}


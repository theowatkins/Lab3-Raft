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
	initCluster(clientCommunicationChannel, createTestPersister())

	for {
		pair := promptForKeyValuePair()
		clientCommunicationChannel <-KeyValue{pair[0], pair[1]}
	}
}

func promptForKeyValuePair() []string {
	pair := keyValuePrompt()
	for len(pair) != 2 {
		fmt.Print("Expected 2 elements received ", len(pair), " elements.")
		pair = keyValuePrompt()
	}
	return pair
}

func keyValuePrompt() []string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter Key-Value pair: K,V: ")
	text, _ := reader.ReadString('\n')
	pair := strings.Split(text, ",")
	return pair
}


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
	applyChannel := make(ApplyChannel)
	fmt.Print("Creating cluster...\n")
	initCluster(clientCommunicationChannel, createTestPersister(), applyChannel)

	for {
		pair := promptForKeyValuePair()
		clientCommunicationChannel <-KeyValue{pair[0], pair[1]}
	}
}

type Raft struct {
	Start func(logEntry LogEntry) (int, int, bool) // returns index, term, isLeader
	GetState func () (int, bool)
}


type ApplyMessage struct {
	term int
	logEntry LogEntry
}

type ApplyChannel = chan ApplyMessage

type NetworkIdentifiers struct {
	voteChannels *[ClusterSize]chan Vote
	appendEntriesCom *[ClusterSize]AppendEntriesCom
	clientCommunicationChannel chan KeyValue
}

func MakeRaft(
	peers NetworkIdentifiers,
	meIndex int,
	persister Persister,
	applyChannel ApplyChannel) Raft {

	previousLogEntries := initializeServerStateFromPersister(persister)
	lastCommittedIndex := len(previousLogEntries)
	currentTerm := 0

	if len(previousLogEntries) > 0 {
		currentTerm = previousLogEntries[len(previousLogEntries)-1].Term
	}

	raftState := ServerState{
		meIndex,
		currentTerm,
		-1,
		previousLogEntries,
	FollowerRole,
	lastCommittedIndex,
	lastCommittedIndex}

	return startServer(
		&raftState,
		peers.voteChannels,
		peers.appendEntriesCom,
		peers.clientCommunicationChannel,
		persister,
		applyChannel)
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
	fmt.Println("Enter Key-Value pair (K,V): ")
	text, _ := reader.ReadString('\n')
	pair := strings.Split(text, ",")
	return pair
}


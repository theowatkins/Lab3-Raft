package main

import "fmt"

func printMessageFromLeader(id int, message AppendEntriesMessage) {
	if debug && len(message.Entries) > 0 {
		printMessage(id, message)
	} else {
		//fmt.Println(id, " received heartbeat from ", message.LeaderId)
	}
}

func printMessage(id int, message AppendEntriesMessage) {
	fmt.Println("Server ", id, " received new entry from ", message.LeaderId, ": ")
	fmt.Println("\tEntries to message: ", message.Entries)
	fmt.Println("\tPrevious index: ", message.PrevLogIndex)
	fmt.Println("\tPrevious term: ", message.PrevLogTerm)
	fmt.Println("\tLeader term: ", message.Term)
}


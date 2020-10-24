package main

import (
	"fmt"
	"math"
)

/* Consistent Hashing Design Decisions
 * 1. Server Identifiers are just int that represent
 *
 */
type Database struct {
	ch *RingHash
	servers * []DatabaseServer
}
const numberOfReplicas = 2

func (db * Database) New (numberOfNodes int) {
	ch := new(RingHash)
	numberVirtualNodesPerServer := 1

	ch.New(numberOfNodes, numberVirtualNodesPerServer)

	//Creates Database
	db.ch = ch

	servers := make([]DatabaseServer, numberOfNodes)

	for serverIndex := 0; serverIndex < numberOfNodes ; serverIndex++ {
		newServer := DatabaseServer{}
		newServer.New(serverIndex)
		servers[serverIndex] = newServer
	}
	db.servers = &servers
}

func (db * Database) AddNode() { //implements CH1
	db.ch.AddNode()
}

func (db * Database) DeleteNode() { //implements CH3
	db.ch.RemoveNode()
}

func (db * Database) Get(key string) int { //implements CH4
	return db.ch.GetAssignedServerForKey(key)
}

func (db * Database) Put(key string, value string)  { //implements CH4
	originalPosition := getKeyPositionOnRing(key)
	serversToSendTo := []int{}
	numberOfCopies := numberOfReplicas + 1
	copyRingDelta := RingCircumference / float64(numberOfCopies)
	for copyIndex := 0; copyIndex < numberOfCopies; copyIndex++ {
		copyPosition := originalPosition + (float64(copyIndex) * copyRingDelta)
		copyPosition = math.Mod(copyPosition, RingCircumference)
		copyAssignedServer := db.ch.GetAssignedServerForPosition(copyPosition)
		serversToSendTo = append(serversToSendTo, copyAssignedServer)
	}

	data := KeyValue{key, value}
	for _, serverIndex := range serversToSendTo {
		(*db.servers)[serverIndex].communicationChannel <- data
	}
}

type DatabaseServer struct {
	serverIndex int
	communicationChannel chan KeyValue
}

func (ds * DatabaseServer) New (serverIndex int) {
	ds.serverIndex = serverIndex
	ds.communicationChannel = make(chan KeyValue)

	//communication channel handler
	go func () {
		for clientKeyValueRequest := range ds.communicationChannel {
			fmt.Println("Database server: ", ds.serverIndex, " received key-value pair: ", clientKeyValueRequest)
		}
	}()
}

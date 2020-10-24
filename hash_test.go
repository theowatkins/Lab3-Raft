package main

import (
	"fmt"
	"math"
	"testing"
	"time"
)

func TestInitialization(t *testing.T) {
	ch := new(RingHash)
	numberOfServers := 1
	numberVirtualNodesPerServer := 1
	ch.New(numberOfServers, numberVirtualNodesPerServer)

	assertEqualInt(numberOfServers, ch.numberOfServers,"numberOfServers", t)
	assertEqualInt(numberVirtualNodesPerServer, ch.virtualNodesPerServer,"numberVirtualNodesPerServer", t)

	expectedPositions := []float64{.5 * RingCircumference, RingCircumference}
	testPositions(expectedPositions, ch, t)
}

func TestAddNode(t *testing.T) {
	ch := new(RingHash)
	numberOfServers := 1
	numberVirtualNodesPerServer := 1
	numberServerPositions := 4
	ch.New(numberOfServers, numberVirtualNodesPerServer)
	ch.AddNode()

	assertEqualInt(numberOfServers + 1, ch.numberOfServers,"numberOfServers", t)
	assertEqualInt(numberVirtualNodesPerServer, ch.virtualNodesPerServer,"numberVirtualNodesPerServer", t)
	assertEqualInt(numberServerPositions, len(ch.serverPositions), "numberServerPositions", t)

	expectedPositions := []float64{
		.25 * RingCircumference,
		.5  * RingCircumference,
		.75 * RingCircumference,
		RingCircumference}
	testPositions(expectedPositions, ch, t)
}

func TestAddNodeTwice(t *testing.T) {
	ch := new(RingHash)
	numberOfServers := 1
	numberVirtualNodesPerServer := 1
	numberServerPositions := 6

	ch.New(numberOfServers, numberVirtualNodesPerServer)
	ch.AddNode()
	ch.AddNode()

	assertEqualInt(numberOfServers + 2, ch.numberOfServers,"numberOfServers", t)
	assertEqualInt(numberVirtualNodesPerServer, ch.virtualNodesPerServer,"numberVirtualNodesPerServer", t)
	assertEqualInt(numberServerPositions, len(ch.serverPositions), "numberServerPositions", t)

	expectedPositions := []float64{
		.25 * RingCircumference,  // 1/8th of circumference
		.375 * RingCircumference, //.375 * Circ
		.5 * RingCircumference,
		.625 * RingCircumference,
		.75 * RingCircumference,
		RingCircumference}
	testPositions(expectedPositions, ch, t)

	expectedServerPositions := []int{1, 2, 0, 2, 1, 0}
	testServerPositions(expectedServerPositions, ch, t)
}

func TestAddNodeTwiceThenRemove(t *testing.T) {
	ch := new(RingHash)
	numberOfServers := 1
	numberVirtualNodesPerServer := 1

	ch.New(numberOfServers, numberVirtualNodesPerServer)
	ch.AddNode()
	ch.AddNode()
	ch.RemoveNode()

	expectedPositions := []float64{
		.25 * RingCircumference, // 1/8th of circumference
		.5 * RingCircumference,
		.75 * RingCircumference,
		RingCircumference}
	testPositions(expectedPositions, ch, t)

	expectedServerPositions := []int{1,  0,  1, 0}
	testServerPositions(expectedServerPositions, ch, t)
}

func TestGetServerWithPosition(t *testing.T) {
	ch := new(RingHash)
	ch.New(1, 1)
	ch.AddNode()
	ch.AddNode()

	assignedServer := ch.GetAssignedServerForPosition(0) //server exists at 5
	assertEqualInt(1, assignedServer, "GetServerWithPosition - clockwise search", t)
	assignedServer = ch.GetAssignedServerForPosition(5) //server exists at 5
	assertEqualInt(0, assignedServer, "GetServerWithPosition - clockwise search", t)
	assignedServer = ch.GetAssignedServerForPosition(.375 * RingCircumference) //athough exact match with 2 captured group goes clockwise
	assertEqualInt(0, assignedServer, "GetServerWithPosition - exact match", t)
}

func TestGetServerWithKey(t *testing.T) {
	ch := new(RingHash)
	ch.New(1, 1)
	ch.AddNode()
	ch.AddNode()

	key := "abc"
	assignedServer := ch.GetAssignedServerForKey(key)
	assertEqualInt(0, assignedServer, "GetServerWithKey", t)
}

func TestDatabaseNew(t *testing.T) {
	db := new(Database)
	db.New(4)

	serversInDatabase := len(*db.servers)
	assertEqualInt(4, serversInDatabase, "serversInDatabase", t)
}

func TestDatabasePut(t *testing.T) {
	db := new(Database)
	db.New(40)
	db.Put("aasdfasdfasbc", "value")
	time.Sleep(time.Second * 2) //allows servers to print their status
}

/* Testing Utility Functions
 *
 */

func testPositions(expected []float64, ch *RingHash, t *testing.T){
	for positionIndex := 0; positionIndex < len(expected); positionIndex++ {
		runIdentifiers := fmt.Sprintf("pos%d", positionIndex)
		position := ch.serverPositions[positionIndex].position
		assertEqualFloat(expected[positionIndex], position, 0.1, runIdentifiers, t)
	}
}

func testServerPositions(expected []int, ch *RingHash, t *testing.T){
	for positionIndex := 0; positionIndex < len(expected); positionIndex++ {
		runIdentifiers := fmt.Sprintf("pos%d", positionIndex)
		serverAtPosition := ch.serverPositions[positionIndex].serverIndex
		assertEqualInt(expected[positionIndex], serverAtPosition,  runIdentifiers, t)
	}
}

func assertEqualInt(a int, b int, key string, t *testing.T) {
	if a != b {
		t.Errorf("%s: expected (%d), got (%d).", key, a, b)
	}
}

func assertEqualFloat(a float64, b float64, delta float64, key string, t *testing.T) {
	if math.Abs(a - b) > delta {
		t.Errorf("%s: expected (%f), got (%f).", key, a, b)
	}
}
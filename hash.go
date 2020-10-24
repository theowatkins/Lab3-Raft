package main

import (
	"math"
	"sort"
)

const RingCircumference = 5.0 //choose prime number so that no factors exists.

/* A servers position on the ring hash.
 *
 */
type ServerPosition struct {
	serverIndex int
	position float64
}

/* Encapsulates data to describe gaps in the ring hash.
 *
 */
type PositionGap struct {
	gapStart float64
	gapEnd float64
	gap float64
}

/* The state of ring hash.
 *
 */
type RingHash struct {
	numberOfServers int
	virtualNodesPerServer int
	serverPositions [] ServerPosition
}

func (ch *RingHash) New(numberOfServers int, virtualNodesPerServer int) RingHash {
	ch.numberOfServers = numberOfServers
	ch.virtualNodesPerServer = virtualNodesPerServer
	ch.serverPositions = make([] ServerPosition, 0)
	createServerPositions(numberOfServers, virtualNodesPerServer, ch)
	return *ch
}

/* Calculates the given key's position in the Ring
 * and returns its clockwise server.
 */
func (ch *RingHash) GetAssignedServerForKey(key string) int {
	keyRingPosition := getKeyPositionOnRing(key)
	return ch.GetAssignedServerForPosition(keyRingPosition)
}

func (ch *RingHash) GetAssignedServerForPosition(position float64) int {
	for _, serverPosition := range ch.serverPositions {
		if serverPosition.position > position {
			return serverPosition.serverIndex
		}
	}
	return 0
}

/* Adds a server (and virtual counterparts) into the biggests gaps in the ring.
 *
 */
func (ch *RingHash) AddNode() {
	newServerIndex := ch.numberOfServers
	totalServersToAdd := 1 + ch.virtualNodesPerServer
	//Add servers to the biggest gaps in the network
	numberOfPositionGaps := len(ch.serverPositions)
	serverGaps := make([]PositionGap, numberOfPositionGaps)
	for serverPositionIndex := 0; serverPositionIndex < numberOfPositionGaps; serverPositionIndex++ {
		nextServerPositionIndex := serverPositionIndex + 1
		if nextServerPositionIndex == numberOfPositionGaps { //if end is reached wrap to beginning
			nextServerPositionIndex = 0
		}
		currentPosition := ch.serverPositions[serverPositionIndex]
		nextPosition := ch.serverPositions[nextServerPositionIndex]

		gap := -1.0
		if nextPosition.position < currentPosition.position { //calculating distance between ring limit
			firstDistance := RingCircumference - currentPosition.position
			secondDistance := nextPosition.position
			gap = firstDistance + secondDistance
		} else {
			gap = math.Abs(nextPosition.position - currentPosition.position)
		}

		serverGaps[serverPositionIndex] = PositionGap{currentPosition.position, nextPosition.position, gap}
	}

	//sort gaps from biggest to smallest gaps
	sort.SliceStable(serverGaps, func(i, j int) bool {
		return serverGaps[i].gap > serverGaps[j].gap
	})

	//Add server in middle of biggest gaps
	for serverCopyIndex := 0; serverCopyIndex < totalServersToAdd; serverCopyIndex++ {
		newGap := serverGaps[serverCopyIndex]
		newServerPositionInRing := math.Mod(newGap.gapStart+(newGap.gap/2), RingCircumference)
		newServerPosition := ServerPosition{newServerIndex, newServerPositionInRing}
		ch.serverPositions = append(ch.serverPositions, newServerPosition)
	}

	ch.numberOfServers++

	sort.SliceStable(ch.serverPositions, func(i, j int) bool {
		return ch.serverPositions[i].position < ch.serverPositions[j].position
	})
}

/* Removes the last server added to the ring.
 *
 */
func (ch *RingHash) RemoveNode() {
	ch.serverPositions = removeItemsContainingServerIndex(ch.serverPositions, ch.numberOfServers - 1)
	ch.numberOfServers--
}

func removeItemsContainingServerIndex(slice []ServerPosition, serverIndexToRemove int) []ServerPosition {
	newPositions := make([]ServerPosition, 0)
	for _, serverPosition := range slice {
		if serverPosition.serverIndex != serverIndexToRemove {
			newPositions = append(newPositions, serverPosition)
		}
	}
	return newPositions
}

func createServerPositions(numberOfServers int, virtualNodesPerServer int, ch *RingHash) {
	totalServers := numberOfServers + (numberOfServers * virtualNodesPerServer)
	spaceBetweenNodes := RingCircumference / float64(totalServers)

	for serverIndex := 0; serverIndex < totalServers; serverIndex++ {
		positionInRing := spaceBetweenNodes * float64(serverIndex+1)
		serverPosition := ServerPosition{serverIndex % numberOfServers, positionInRing}
		ch.serverPositions = append(ch.serverPositions, serverPosition)
	}

	sort.SliceStable(ch.serverPositions, func(i, j int) bool {
		return ch.serverPositions[i].position < ch.serverPositions[j].position
	})
}

/* Adds the ascii characters values in given key and finds its position in the ring
 * by taking the remainder after dividing by RingCircumference. Note, because we are
 * always dividing an integer the RingCircumference cannot be 1.
 */
func getKeyPositionOnRing(key string) float64 {
	hash := getKeyHash(key)
	positionOnRing := math.Mod(float64(hash), RingCircumference) //Notes,
	return positionOnRing
}

func getKeyHash(key string) int {
	hash := 0
	for _, char := range key {
		hash += int(char)
	}
	return hash
}

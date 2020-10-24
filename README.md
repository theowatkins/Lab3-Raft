System Name: Lab 3 - Raft and Consistent Hashing
Authors: Theo Watkins and Alberto Rodriguez

************************************************************************************************************

# System Design

Our system is a command-line interface that accepts (key,value) pairs and appends them to the system’s log. This log is persistent via a Persister object described in the specification, currently it is a write-to-disk persister. Our system immediately starts elections and will restart elections if a leader is not sending heartbeats. We have tested that servers with logs that get behind are brought up-to-date as soon as they resume. We have also tested that the log persists between runs and that consensus can be reached even when some servers are down. 

Our implementation of raft makes use of go channels instead of RPCs. However, go channel closely mimics the RPC call specifications, using structs with matching properties to communicate. Also, we attempted to follow the interface closely with the following relationships:

Make -> (raft.go, 47)
Raft.Start -> (cluster.go, 61)
Raft.GetState -> (cluster.go, 68)

************************************************************************************************************

# Using the command line interface

Our Raft implementation requires a Linux OS with Go downloaded.  The code can be run using the following command:

$ ./start.sh

Then the program will spawn a simulation of Raft with 8 servers in the cluster.  The program will then prompt for command line input (“Enter Key-Value pair K,V: “).  

NOTE -  This prompt may not display in an intuitive location because we have added some print statements to display the status of the system that fire asynchronously.  There are 3 things that are printed: the leader when one is elected, the current log of a server after it processes a non-empty appendEntry request, and the size of the log after a new entry is committed to the persistent state.  

As key-value pairs are entered, feedback is printed and pairs are added to the logs of each server (in a LogEntry object containing the term and the key-value pair).  A good way to see some interesting output is to enter a few key-value pairs and then type Ctrl^C.  The state of the system will then be saved to TestLog.csv and when the system is restarted it will pick up where it left off. New entries committed will be in a new term, but the entries from previous terms will  still be present on all servers.  

************************************************************************************************************

# Consistent Hashing

Consistent hashing is implemented in a separate system that is able to create and delete nodes in the hashing schema. Initially, servers are given a constant number of virtual nodes who are placed around the hash ring or circle. RingHash (hash.go:30) struct keeps track of the positions of the servers and their virtual counterparts. At initialization all real and virtual servers are spaced equally around the ring. Subsequent server additions (both real and virtual) are placed in between the biggest gaps in the system. This allows for the system to dynamically place new nodes in the areas that need it the most. For each key-value pair stored a constant number of replicas (2 by default) are stored with it on different servers selected from equally spaced locations on the ring. Our recovery strategy leverages the replicas to deal with servers that are failed. We use the log to record where replicas are sent and then we access an entry we would try the primary server, using the replicas as backups in case it is down.

************************************************************************************************************

# Requirement Implementations

We read over the specification for Lab 3 and the Raft paper to come up with a list of requirements, located in requirements.txt. For each entry below, each requirements id is placed next to the locations in the system that implement it.

## AppendEntries (AE):

AE1, (cluster.go, 159)
AE2, (cluster.go, 161)
AE3, (cluster.go, 163)
AE4, (cluster.go, 163)
AE5, (cluster.go, 168)

## RequestVote (RV):

RV1, (election.go 58)
RV2, (election.go, 60), (election.go, 70) 

## Requirements for Servers :

AS1, This requirement is not implemented because our design did not need to use the lastApplied variable.  We calculate lastApplied from the current state of a servers log, and ensure safe indexing whenever accessing items in logs.  
AS2, 
FUNCTION: (cluster.go, 133)
CALLS: (cluster.go, 155), (leader.go, 155), (election.go, 54), (election.go, 105)


## Followers (F):

F1, (election.go, 52)
F2, (election.go, 26)

## Candidates (C):

C1, (election.go, 27)
C2, (election.go, 111)
C3, (election.go, 123)
C4, (cluster.go, 88)

## Leaders (L):

L1, (leader.go, 39)
L2, (leader.go, 112-122)
L3,(leader.go, 171-183)
L4, (leader.go, 85-125)


## Persistent State Requirements:

P1, (raft.go, 53) (leader.go, 130)
P2, (raft.go, 50)
P3, (raft.go, 53)

## Consistent Hashing Requirements (CH):

CH1, (database.go, 37)
CH2, (database.go, 41)
CH3, (database.go, 45)
CH4, (database.go, 49)





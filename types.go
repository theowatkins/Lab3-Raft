package main

// KeyValues are sent from the client
// to the cluster for storage.  
type KeyValue struct {
	Key string
	Value string
}

// LogEntries are sent from the leader 
// to the followers
type LogEntry struct {
    Idx int
    Term int
    Content KeyValue
}

type ServerRole string
const LeaderRole ServerRole = "LeaderRole"
const FollowerRole ServerRole = "FollowerRole"
const CandidateRole ServerRole = "CandidateRole"

// ServerStates store the id, log, and
// Role  of a server (leader, follower, or candidate)
type ServerState struct {
	ServerId    int
	CurrentTerm int
	VotedFor    int
	Log         []LogEntry
	Role        ServerRole
}
}

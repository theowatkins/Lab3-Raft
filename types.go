// types.go

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

// ServerStates store the id, log, and
// status of a server (leader or follower)
type ServerState struct {
	ServerId int
	Leader int // ServerId of leader
	Log []LogEntry
}
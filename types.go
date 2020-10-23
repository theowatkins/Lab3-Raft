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
    Term int
    Content KeyValue
}

type AppendEntriesMessage struct {
	/* Leaders term
	 *
	 */
	Term int

	/* subject to change but used for redirection to leader
	 *
	 */
	LeaderId int

	/* index of log entry immediately preceding new ones
	 *
	 */
	PrevLogIndex int

	/* term of PrevLogIndex entry
	 *
	 */
	PrevLogTerm int

	/* log entries to store
	 *
	 */
	Entries []LogEntry

	/* leader's commitIndex
	 *
	 */
	LeaderCommit int
}

type AppendEntriesResponse struct {
	/* Current Term. Used for leader to update itself.
	 *
	 */
	term int

	/* True if follower contained entry matching prevLogIndex and prevLogTerm
	 * False otherwise.
	 */
	success bool

	/* Message to retry sending (with more log entries) if failed
	 * 
	 */
	message AppendEntriesMessage
}

type AppendEntriesCom struct {
	/* Messages from client to cluster
	 *
	 */
	message chan AppendEntriesMessage

	/* channel for response from the cluster
	 *
	 */
	response chan AppendEntriesResponse
} 

type ServerRole string
const LeaderRole ServerRole = "LeaderRole"
const FollowerRole ServerRole = "FollowerRole"
const CandidateRole ServerRole = "CandidateRole"

type ServerState struct {
	ServerId    int
	/* Latest term server has seen.
	 * Initialized to 0 on first boot, increases monotonically.
	 */
	CurrentTerm int

	/* CandidateId that received vote in current term.
	 * Or nullif none.
	 */
	VotedFor    int

	/* Log Entries; each entry contains command for state machine, and term when entry was received by leader.
	 * First index is 1.
	 */
	Log         []LogEntry

	/* Enum containing the current role of the server. Used to block candidates
	 * from becoming leaders if leader already exists.
	 */
	Role        ServerRole

	/* Index of highest log entry known to be committed.
	 * Initialized to 0, increases monotonically.
	 */
	commitIndex int

	/* Index of highest log entry applied to state.
	 * Initialized to 0, increases monotonically.
	 */
	lastApplied int
}

type ServerTermState struct {
	/* For each server, index of the next log entry to send to that server.
	 * Initialized to leader last log index + 1.
	 */
	nextIndex int

	/* For each server, index of highest log entry known to be replicated on server.
	 * Initialized to 0, increases monotonically.
	 */
	matchIndex int
}

type Vote struct {
	Term int
	VoteFor int
	Responses chan VoteResponse
	LastLogIndex int
	LastLogTerm int
}

type VoteResponse struct {
	GotVote bool
	Term int
}

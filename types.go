//types.go

type KeyValue struct {
	Key string
	Value string
}

type LogEntry struct {
    idx int
    term int
    content KeyValue
}
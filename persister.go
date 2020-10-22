package main

import "fmt"

/* Had a lot of trouble finding persister.go libary. This is the as close as I got:
 * https://godoc.org/github.com/nedscode/memdb/persist#LoadFunc
 *
 * I couldn't download it but I imagined it would look something like this:
 */
type LoadFunc func(id string, indexer interface{})

type Persister interface {
	// Save is called to request persistent save of the indexer with id
	Save(id string, entry LogEntry) error

	// Load is called at create time to load all of the persisted items and call loadFunc with each
	Load(loadFunc LoadFunc) error

	// Remove is called when an indexer is expired or deleted and needs removal from persistent store
	Remove(id string) error
}

func initializeServerStateFromPersister(persister Persister) []LogEntry {
	var logEntries []LogEntry
	loadFunc := func(id string, logEntry interface{}){
		//add logEntry to logEntries
	}
	loadError := persister.Load(loadFunc)
	if loadError != nil {
		fmt.Printf("Error occurred while loading persistent state.")
	}
	return logEntries
}

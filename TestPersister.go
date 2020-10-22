package main

import "fmt"

type TestPersister struct {
	entriesSinceStart * []LogEntry
}

func (t TestPersister) Save(id string, logEntry LogEntry) error {
	*t.entriesSinceStart = append(*t.entriesSinceStart, logEntry)
	fmt.Println("saving log entry to persister. New total entries: ", len(*t.entriesSinceStart))
	return nil
}

func (t TestPersister) Load(loadFunc LoadFunc) error {
	fmt.Println("loading from persister...")
	return nil
}

func (t TestPersister) Remove(id string) error {
	panic("implement me")
}

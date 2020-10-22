package main

import (
	"fmt"
)

const TestLogFileName = "TestLog.csv"

type TestPersister struct {
	inMemoryLog * []LogEntry
}

func createTestPersister() TestPersister {
	var logBuffer []LogEntry
	persister := TestPersister{}
	persister.inMemoryLog = &logBuffer
	return persister
}

/* Adds a line csv row to the log corresponding to given log entry.
 *
 */
func (t TestPersister) Save(id string, logEntry LogEntry) error {
	*t.inMemoryLog = append(*t.inMemoryLog, logEntry)
	addEntryToFile(TestLogFileName, logEntryAsCVSEntry(logEntry))
	fmt.Println("saving log entry to persister. New entries this session: ", len(*t.inMemoryLog))

	return nil
}

/* Reads CSV log and creates log entry per file.
 *
 */
func (t TestPersister) Load(loadFunc LoadFunc) error {
	*t.inMemoryLog = readLogEntryCSVFile(TestLogFileName)
	fmt.Println("Loaded ", len(*t.inMemoryLog), " many entries from persistent state.")
	return nil
}

/* Removes a line in the log given the id of the entry.
 *
 */
func (t TestPersister) Remove(id string) error {
	panic("implement me")
}




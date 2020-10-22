package main

import (
	"fmt"
	"strconv"
)

/* TestPersister is a testing library that writes to a file in a csv format.
 * Each line corresponds to a LogEntry. Each line has three values"
 * 1. Term
 * 2. Key
 * 3. Value
 */
const TestLogFileName = "TestLog.csv" //the name of the log

type TestPersister struct {
	inMemoryLog * []LogEntry //used to load and store list during normal operation. Used mostly for testing purposes.
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
	addEntryToCSVFile(TestLogFileName, logEntryAsCVSEntry(logEntry))
	fmt.Println("saving log entry to persister. New entries this session: ", len(*t.inMemoryLog))

	return nil
}

/* Reads CSV log and creates log entry per file.
 *
 */
func (t TestPersister) Load(loadFunc LoadFunc) error {
	loadedLogEntries := readLogEntriesFromCSVFile(TestLogFileName)
	*t.inMemoryLog = loadedLogEntries
	for logEntryIndex, logEntry := range loadedLogEntries {
		loadFunc(strconv.Itoa(logEntryIndex), logEntry)
	}
	fmt.Println("Loaded ", len(*t.inMemoryLog), " many entries from persistent state.")
	return nil
}

/* Removes a line in the log given the id of the entry.
 *
 */
func (t TestPersister) Remove(id string) error {
	panic("implement me")
}




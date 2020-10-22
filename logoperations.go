package main

import (
	"encoding/csv"
	"io"
	"log"
	"os"
	"strconv"
)

const NumberOfFieldsInLogEntry = 3

/* Responsible for:
 * 1. Converting between LogEntry and format in log
 * 2. Adding LogEntry to log
 * 3. Reading and parsing log entry
 */

func logEntryAsCVSEntry(logEntry LogEntry) []string {
	return []string{strconv.Itoa(logEntry.Term), logEntry.Content.Key, logEntry.Content.Value}
}

func CSVEntryAsLogEntry(logEntry []string) LogEntry {
	if len(logEntry) != NumberOfFieldsInLogEntry {
		log.Fatal("Row had too few or too many values:", len(logEntry))
	}
	term, err := strconv.Atoi(logEntry[0])
	checkError("Entry term number could not parsed into a number: ", err)

	logEntryContent := KeyValue{logEntry[1], logEntry[2]}
	logEntryParsed := LogEntry{term, logEntryContent}
	return logEntryParsed
}

func readLogEntriesFromCSVFile(fileName string) []LogEntry {
	var logEntries []LogEntry
	csvFile, err := os.Open(fileName)

	checkError("Couldn't open the csv file", err)
	r := csv.NewReader(csvFile)

	// Iterate through the records
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		checkError("Error reading record in file:", err)

		logEntry := CSVEntryAsLogEntry(record)
		logEntries = append(logEntries, logEntry)
	}
	return logEntries
}

func addEntryToCSVFile(fileName string, row []string) {
	file, err := os.OpenFile(fileName,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	checkError("Error opening file:", err)
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	err = writer.Write(row)
	checkError("Cannot write to file", err)
}

func checkError(message string, err error) {
	if err != nil {
		log.Fatal(message, err)
	}
}
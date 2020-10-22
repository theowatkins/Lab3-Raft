package main

import (
	"encoding/csv"
	"io"
	"log"
	"os"
	"strconv"
)

func logEntryAsCVSEntry(logEntry LogEntry) []string {
	return []string{strconv.Itoa(logEntry.Term), logEntry.Content.Key, logEntry.Content.Value}
}

func readLogEntryCSVFile(fileName string) []LogEntry {
	var logEntries []LogEntry
	// Open the file
	csvFile, err := os.Open(fileName)
	if err != nil {
		log.Fatalln("Couldn't open the csv file", err)
	}

	// Parse the file
	r := csv.NewReader(csvFile)
	//r := csv.NewReader(bufio.NewReader(csvFile))

	// Iterate through the records
	for {
		// Read each record from csv
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		term, err := strconv.Atoi(record[0])
		key := record[1]
		value := record[2]
		logEntryContent := KeyValue{key, value}
		logEntry := LogEntry{term, logEntryContent}
		logEntries = append(logEntries, logEntry)
	}
	return logEntries
}

func addEntryToFile(fileName string, row []string) {
	file, err := os.OpenFile(fileName,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}

	checkError("Cannot create file", err)
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

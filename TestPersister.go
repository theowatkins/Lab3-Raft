package main

import "fmt"

type TestPersister struct {

}

func (t TestPersister) Save(id string, logEntry interface{}) error {
	fmt.Println("saving log entry to persister: ", logEntry)
	return nil
}

func (t TestPersister) Load(loadFunc LoadFunc) error {
	fmt.Println("loading from persister...")
	return nil
}

func (t TestPersister) Remove(id string) error {
	panic("implement me")
}

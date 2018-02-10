package storage

import (
	"bytes"
	"encoding/gob"

	"fmt"
)

// TaskAttributes is a struct which is used to transfer data from/to stores.
// All task data are converted from/to string to prevent the store from
// worrying about details of converting data to the proper formats.
type TaskAttributes struct {
	Hash        string
	//task.Func.Name
	Name        string
	LastRun     string
	NextRun     string
	Duration    string
	IsRecurring string
	Params      string
	VideoFileName	string
}

// TaskStore is the interface to implement when adding custom task storage.
type TaskStore interface {
	Add(TaskAttributes) error

	AddJobStatus(string) error

	Fetch() ([]TaskAttributes, error)

	Remove(TaskAttributes) error
}
// NewFromBytes returns a Job instance from a byte representation.
func NewFromBytes(b []byte) (*TaskAttributes, error) {
	j := &TaskAttributes{}

	buf := bytes.NewBuffer(b)
	err := gob.NewDecoder(buf).Decode(&j)
	if err != nil {
		return nil, err
	}

	return j, nil
}
// Bytes returns the byte representation of the Job.
func (j TaskAttributes) Bytes() ([]byte, error) {
	buff := new(bytes.Buffer)
	enc := gob.NewEncoder(buff)
	err := enc.Encode(j)
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}
// ErrJobNotFound is raised when a Job is able to be found within a database.
type ErrJobNotFound string

func (id ErrJobNotFound) Error() string {
	return fmt.Sprintf("Job with id of %s not found.", string(id))
}
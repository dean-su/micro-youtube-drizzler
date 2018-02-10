package storage

import (
	log "github.com/Sirupsen/logrus"
	"github.com/garyburd/redigo/redis"
)

var (
	// HashKey is the hash key where jobs are persisted.
	HashKey = "youtubejobs"
	HashVideoKey = "videoStatus"
)

// DB is concrete implementation of the JobDB interface, that uses Redis for persistence.
type DB struct {
	conn      redis.Conn
	keyprefix string
}

// New instantiates a new DB.
func NewRedis(address string, password redis.DialOption, sendPassword bool) *DB {
	var conn redis.Conn
	var err error
	if address == "" {
		address = "127.0.0.1:6379"
	}
	if sendPassword {
		conn, err = redis.Dial("tcp", address, password)
	} else {
		conn, err = redis.Dial("tcp", address)
	}
	if err != nil {
		log.Fatal(err)
	}
	return &DB{
		conn: conn,
	}
}

// GetAll returns all persisted Jobs.
func (d DB) Fetch() ([]TaskAttributes, error) {
	jobs := []TaskAttributes{}
	//var jobs []TaskAttributes
	vals, err := d.conn.Do("HVALS", HashKey)
	if err != nil {
		return jobs, err
	}

	for _, val := range vals.([]interface{}) {

		j, err := NewFromBytes(val.([]byte))
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, *j)
	}

	return jobs, nil
}

// Get returns a persisted Job.
func (d DB) Get(id string) (TaskAttributes, error) {
	val, err := d.conn.Do("HGET", HashKey, id)
	if err != nil {
		return TaskAttributes{}, err
	}
	if val == nil {
		return TaskAttributes{}, ErrJobNotFound(id)
	}
	j , err := NewFromBytes(val.([]byte))
	return *j, err
}

// Delete deletes a persisted Job.
func (d DB) Remove(task TaskAttributes) error {
	_, err := d.conn.Do("HDEL", HashKey, task.Hash)
	if err != nil {
		return err
	}

	return nil
}

// Save persists a Job.
func (d DB) Add(j TaskAttributes) error {
	bytes, err := j.Bytes()
	if err != nil {
		return err
	}

	_, err = d.conn.Do("HSET", HashKey, j.Hash, bytes)
	if err != nil {
		return err
	}

	return nil
}
// Save persists a Job Status.
func (d DB) AddJobStatus(s string) error {

	_, err := d.conn.Do("HSET", HashVideoKey, s, "P")
	if err != nil {
		return err
	}
	return nil
}

// Close closes the connection to Redis.
func (d DB) Close() error {
	err := d.conn.Close()
	if err != nil {
		return err
	}
	return nil
}

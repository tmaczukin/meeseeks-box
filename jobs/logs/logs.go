package logs

import (
	"bytes"
	"errors"
	"fmt"

	bolt "github.com/coreos/bbolt"
	"github.com/gomeeseeks/meeseeks-box/db"
)

var logsBucketKey = []byte("logs")
var errorKey = []byte("error")

// ErrNoLogsForJob is returned when we try to extract the logs of a non existing job
var ErrNoLogsForJob = errors.New("No logs for job")

// JobLog represents all the logging information of a given Job
type JobLog struct {
	Error  string
	Output string
}

// GetError returns nil or an error depending on the current JobLog setup
func (j JobLog) GetError() error {
	if j.Error == "" {
		return nil
	}
	return errors.New(j.Error)
}

// Append adds a new line to the logs of the given Job
func Append(jobID uint64, content string) error {
	if content == "" {
		return nil
	}
	return db.Update(func(tx *bolt.Tx) error {
		jobBucket, err := getJobBucket(jobID, tx)
		if err != nil {
			return fmt.Errorf("could not get job %d bucket: %s", jobID, err)
		}

		sequence, err := jobBucket.NextSequence()
		if err != nil {
			return fmt.Errorf("could not get next sequence for job %d: %s", jobID, err)
		}

		return jobBucket.Put(db.IDToBytes(sequence), []byte(content))
	})
}

// SetError sets the error message for the given Job
func SetError(jobID uint64, jobErr error) error {
	if jobErr == nil {
		return nil
	}
	return db.Update(func(tx *bolt.Tx) error {
		jobBucket, err := getJobBucket(jobID, tx)
		if err != nil {
			return fmt.Errorf("could not get job %d bucket: %s", jobID, err)
		}
		errorBucket, err := jobBucket.CreateBucketIfNotExists(errorKey)
		if err != nil {
			return fmt.Errorf("could not get error bucket for job %d: %s", jobID, err)
		}

		return errorBucket.Put(errorKey, []byte(jobErr.Error()))
	})
}

// Get returns the JobLog for the given Job
func Get(jobID uint64) (JobLog, error) {
	job := &JobLog{}
	err := db.View(func(tx *bolt.Tx) error {
		logsBucket := tx.Bucket(logsBucketKey)
		if logsBucket == nil {
			return ErrNoLogsForJob
		}

		jobBucket := logsBucket.Bucket(db.IDToBytes(jobID))
		if jobBucket == nil {
			return ErrNoLogsForJob
		}

		c := jobBucket.Cursor()
		_, line := c.First()
		out := bytes.NewBufferString("")
		for {
			if line == nil {
				break
			}
			out.Write(line)
			_, line = c.Next()
		}
		job.Output = out.String()

		errorBucket := jobBucket.Bucket(errorKey)
		if errorBucket != nil {
			job.Error = string(errorBucket.Get(errorKey))
		}
		return nil
	})
	return *job, err
}

func getJobBucket(jobID uint64, tx *bolt.Tx) (*bolt.Bucket, error) {
	logsBucket, err := tx.CreateBucketIfNotExists(logsBucketKey)
	if err != nil {
		return nil, fmt.Errorf("could not get logs bucket: %s", err)
	}
	return logsBucket.CreateBucketIfNotExists(db.IDToBytes(jobID))

}

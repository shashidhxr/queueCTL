package models

import "time"

type JobState string

const (
    StatePending    JobState = "pending"
    StateProcessing JobState = "processing"
    StateCompleted  JobState = "completed"
    StateFailed     JobState = "failed"
    StateDead       JobState = "dead"
)

type Job struct {
	ID         string    `json:"id"`
	Command    string    `json:"command"`
	State      JobState  `json:"state"`
	Attempts   int       `json:"attempts"`
	MaxRetries int       `json:"max_retries"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	Error      string    `json:"error,omitempty"`
	NextRetry  *time.Time `json:"next_retry,omitempty"`
}
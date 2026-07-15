package models

import "time"

// JobStatus is the lifecycle state of an ImportJob: pending -> processing
// -> (completed | failed).
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// ImportJob tracks the lifecycle of one asynchronous Excel import so the
// client can poll for progress instead of blocking on a long HTTP request.
type ImportJob struct {
	ID           string     `json:"id"`
	Status       JobStatus  `json:"status"`
	FileName     string     `json:"file_name"`
	TotalRows    int        `json:"total_rows"`
	InsertedRows int        `json:"inserted_rows"`
	FailedRows   int        `json:"failed_rows"`
	Errors       []string   `json:"errors,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

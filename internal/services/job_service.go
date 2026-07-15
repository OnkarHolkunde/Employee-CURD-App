package services

import (
	"sync"
	"time"

	"excel-crud-app/internal/models"

	"github.com/google/uuid"
)

// JobStore is a thread-safe in-memory registry of import jobs, letting the
// upload endpoint return immediately while a goroutine processes in the background.
type JobStore struct {
	mu   sync.RWMutex
	jobs map[string]*models.ImportJob
}

func NewJobStore() *JobStore {
	return &JobStore{
		jobs: make(map[string]*models.ImportJob),
	}
}

// CreateJob registers a new import job in JobStatusPending and returns it.
func (s *JobStore) CreateJob(fileName string) *models.ImportJob {
	job := &models.ImportJob{
		ID:        uuid.NewString(),
		Status:    models.JobStatusPending,
		FileName:  fileName,
		CreatedAt: time.Now(),
	}
	s.mu.Lock()
	s.jobs[job.ID] = job
	s.mu.Unlock()
	return job
}

// Get looks up a job by ID, comma-ok style.
func (s *JobStore) Get(id string) (*models.ImportJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	job, ok := s.jobs[id]
	return job, ok
}

// Update applies mutate to the job with the given ID under the write
// lock. A no-op if the ID doesn't exist.
func (s *JobStore) Update(id string, mutate func(job *models.ImportJob)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, ok := s.jobs[id]; ok {
		mutate(job)
	}
}

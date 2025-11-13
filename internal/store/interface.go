package store

import (
	"context"

	"github.com/shashidhxr/queueCTL/pkg/models"
)

type store interface {
	// Job operations
    SaveJob(ctx context.Context, job *models.Job) error
    GetJob(ctx context.Context, id string) (*models.Job, error)
    UpdateJobState(ctx context.Context, id string, state models.JobState) error
    ListJobs(ctx context.Context, state models.JobState) ([]*models.Job, error)
    GetPendingJobs(ctx context.Context) ([]*models.Job, error)
    
    // Worker operations
    AcquireJob(ctx context.Context) (*models.Job, error)
    
    // DLQ operations
    GetDLQJobs(ctx context.Context) ([]*models.Job, error)
    RetryDLQJob(ctx context.Context, jobID string) error
    
    // Config operations
    GetConfig(ctx context.Context) (*models.Config, error)
    SetConfig(ctx context.Context, config *models.Config) error
    
    // Status
    GetJobStats(ctx context.Context) (map[models.JobState]int, error)
    
    Close() error
	
}
package store

import (
	"context"

	"github.com/shashidhxr/queueCTL/pkg/models"
)

type store interface {
	SaveJob(ctx context.Context, job *models.Job) error
	GetJob(ctx context.Context, id string) (*models.Job, error)
	// UpdateJob(ctx context.Context, id string, state models.JobState) error

	AcquireJob(ctx context.Context) (*models.Job, error)

	GetJobStatus(ctx context.Context) (map[models.JobState]int, error)

	Close() error
}
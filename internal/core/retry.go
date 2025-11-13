package core

import (
	"math"
	"time"

	"github.com/shashidhxr/queueCTL/internal/store"
)

type RetryManager struct {
	storage     store.SQLiteStorage
	backoffBase float64
}

func NewRetryManager(store store.SQLiteStorage) *RetryManager {
	return &RetryManager{
		storage:     store,
		backoffBase: 2.0, // default
	}
}

func (rm *RetryManager) CalculateBackoff(attempts int) time.Duration {
	seconds := math.Pow(rm.backoffBase, float64(attempts))
	maxDelay := 300.0
	if seconds > maxDelay { // cap at 5 minutes
		seconds = maxDelay
	}
	return time.Duration(seconds) * time.Second
}

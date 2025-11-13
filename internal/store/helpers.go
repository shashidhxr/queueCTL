package store

import "time"

func nullableTime(t *time.Time) any {
	if t == nil { return nil }
	return *t
}

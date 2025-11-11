package store

import (
	"database/sql"
	"fmt"
	"os"
	"time"
)

type Store interface {
	Init() error
}

type sqliteStore struct {
	db *sql.DB
}

func (s *sqliteStore) Init() error {
	// jobTable
}
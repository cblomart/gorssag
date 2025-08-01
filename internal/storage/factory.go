package storage

import "gorssag/internal/config"

// NewStorage creates a new SQLite storage instance
func NewStorage(dataDir string, cfg *config.Config) (Storage, error) {
	return NewSQLiteStorage(dataDir, cfg)
}

package storage

// NewStorage creates a new SQLite storage instance
func NewStorage(dataDir string) (Storage, error) {
	return NewSQLiteStorage(dataDir)
}

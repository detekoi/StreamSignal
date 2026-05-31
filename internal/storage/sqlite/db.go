package sqlite

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const (
	driverName = "sqlite"
	dsnOptions = "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
)

func Open(path string) (*sql.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite path is required")
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve sqlite path: %w", err)
	}

	db, err := sql.Open(driverName, absolutePath+dsnOptions)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite database: %w", err)
	}

	if err := ApplyMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func OpenInMemory() (*sql.DB, error) {
	db, err := sql.Open(driverName, "file::memory:?cache=shared")
	if err != nil {
		return nil, fmt.Errorf("open in-memory sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping in-memory sqlite database: %w", err)
	}

	if err := ApplyMigrations(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

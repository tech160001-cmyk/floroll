package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	database, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := database.Ping(); err != nil {
		database.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	database.SetMaxOpenConns(1)

	return database, nil
}

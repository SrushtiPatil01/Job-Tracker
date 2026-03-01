package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

// Connect opens a connection to PostgreSQL and runs migrations.
func Connect(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("Database connected and migrated")
	return db, nil
}

func migrate(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS applications (
		id            VARCHAR(36) PRIMARY KEY,
		company       VARCHAR(255) NOT NULL,
		role          VARCHAR(255) NOT NULL,
		status        VARCHAR(50)  NOT NULL DEFAULT 'Applied',
		applied_at    TIMESTAMP    NOT NULL,
		s3_object_key VARCHAR(512) DEFAULT '',
		created_at    TIMESTAMP    NOT NULL DEFAULT NOW(),
		updated_at    TIMESTAMP    NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_applications_status ON applications(status);
	CREATE INDEX IF NOT EXISTS idx_applications_applied_at ON applications(applied_at);
	`
	_, err := db.Exec(query)
	return err
}
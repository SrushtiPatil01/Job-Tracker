package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/SrushtiPatil01/Job-Tracker/internal/models"
)

// InsertApplication creates a new application record.
func InsertApplication(db *sql.DB, app *models.Application) error {
	query := `
		INSERT INTO applications (id, company, role, status, applied_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := db.Exec(query, app.ID, app.Company, app.Role, app.Status, app.AppliedAt, app.CreatedAt, app.UpdatedAt)
	return err
}

// UpdateApplicationStatus changes the status of an existing application.
func UpdateApplicationStatus(db *sql.DB, id, status string) error {
	query := `UPDATE applications SET status = $1, updated_at = $2 WHERE id = $3`
	res, err := db.Exec(query, status, time.Now(), id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("application %s not found", id)
	}
	return nil
}

// UpdateS3ObjectKey stores the S3 key for a given application.
func UpdateS3ObjectKey(db *sql.DB, id, key string) error {
	query := `UPDATE applications SET s3_object_key = $1, updated_at = $2 WHERE id = $3`
	_, err := db.Exec(query, key, time.Now(), id)
	return err
}

// GetApplicationByID retrieves a single application.
func GetApplicationByID(db *sql.DB, id string) (*models.Application, error) {
	query := `SELECT id, company, role, status, applied_at, s3_object_key, created_at, updated_at FROM applications WHERE id = $1`
	row := db.QueryRow(query, id)

	var app models.Application
	err := row.Scan(&app.ID, &app.Company, &app.Role, &app.Status, &app.AppliedAt, &app.S3ObjectKey, &app.CreatedAt, &app.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// ListApplications returns all applications, optionally filtered by status.
func ListApplications(db *sql.DB, status string) ([]models.Application, error) {
	var rows *sql.Rows
	var err error

	if status != "" {
		query := `SELECT id, company, role, status, applied_at, s3_object_key, created_at, updated_at FROM applications WHERE status = $1 ORDER BY applied_at DESC`
		rows, err = db.Query(query, status)
	} else {
		query := `SELECT id, company, role, status, applied_at, s3_object_key, created_at, updated_at FROM applications ORDER BY applied_at DESC`
		rows, err = db.Query(query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []models.Application
	for rows.Next() {
		var app models.Application
		if err := rows.Scan(&app.ID, &app.Company, &app.Role, &app.Status, &app.AppliedAt, &app.S3ObjectKey, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}

// GetStats computes application statistics.
func GetStats(db *sql.DB) (*models.StatsResponse, error) {
	rows, err := db.Query(`SELECT status, COUNT(*) FROM applications GROUP BY status`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byStage := make(map[string]int)
	total := 0
	appliedCount := 0

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		byStage[status] = count
		total += count
		if status == models.StatusApplied {
			appliedCount = count
		}
	}

	responseRate := 0.0
	if total > 0 {
		responseRate = float64(total-appliedCount) / float64(total) * 100
	}

	return &models.StatsResponse{
		Total:        total,
		ResponseRate: responseRate,
		ByStage:      byStage,
	}, rows.Err()
}

// GetPendingReminders returns applications still in "Applied" status
// where applied_at is older than the given threshold.
func GetPendingReminders(db *sql.DB, olderThan time.Time) ([]models.Application, error) {
	query := `
		SELECT id, company, role, status, applied_at, s3_object_key, created_at, updated_at
		FROM applications
		WHERE status = 'Applied' AND applied_at < $1
		ORDER BY applied_at ASC
	`
	rows, err := db.Query(query, olderThan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []models.Application
	for rows.Next() {
		var app models.Application
		if err := rows.Scan(&app.ID, &app.Company, &app.Role, &app.Status, &app.AppliedAt, &app.S3ObjectKey, &app.CreatedAt, &app.UpdatedAt); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, rows.Err()
}
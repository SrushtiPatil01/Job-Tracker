package models

import "time"

// Valid application statuses.
const (
	StatusApplied   = "Applied"
	StatusOA        = "OA"
	StatusInterview = "Interview"
	StatusOffer     = "Offer"
	StatusRejected  = "Rejected"
)

// ValidStatuses is the set of allowed status values.
var ValidStatuses = map[string]bool{
	StatusApplied:   true,
	StatusOA:        true,
	StatusInterview: true,
	StatusOffer:     true,
	StatusRejected:  true,
}

// Application represents a single job application record.
type Application struct {
	ID          string    `json:"id"`
	Company     string    `json:"company"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	AppliedAt   time.Time `json:"applied_at"`
	S3ObjectKey string    `json:"s3_object_key,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateApplicationRequest is the expected body for POST /applications.
type CreateApplicationRequest struct {
	Company   string `json:"company"   binding:"required"`
	Role      string `json:"role"      binding:"required"`
	AppliedAt string `json:"applied_at"` // optional, defaults to now
}

// UpdateApplicationRequest is the expected body for PATCH /applications/:id.
type UpdateApplicationRequest struct {
	Status string `json:"status" binding:"required"`
}

// StatsResponse is returned by GET /applications/stats.
type StatsResponse struct {
	Total        int            `json:"total"`
	ResponseRate float64        `json:"response_rate"`
	ByStage      map[string]int `json:"by_stage"`
}

// ApplicationCreatedEvent is published to Kafka on new applications.
type ApplicationCreatedEvent struct {
	ApplicationID string    `json:"application_id"`
	AppliedAt     time.Time `json:"applied_at"`
}
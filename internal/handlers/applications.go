package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/SrushtiPatil01/Job-Tracker/internal/database"
	"github.com/SrushtiPatil01/Job-Tracker/internal/kafka"
	"github.com/SrushtiPatil01/Job-Tracker/internal/models"
	"github.com/SrushtiPatil01/Job-Tracker/internal/s3"
)

// Handler holds shared dependencies for all route handlers.
type Handler struct {
	DB       *sql.DB
	Producer *kafka.Producer
	S3Client *s3.Client
}

// CreateApplication handles POST /applications.
func (h *Handler) CreateApplication(c *gin.Context) {
	var req models.CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse applied_at or default to now.
	appliedAt := time.Now()
	if req.AppliedAt != "" {
		parsed, err := time.Parse("2006-01-02", req.AppliedAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "applied_at must be YYYY-MM-DD format"})
			return
		}
		appliedAt = parsed
	}

	app := &models.Application{
		ID:        uuid.New().String(),
		Company:   req.Company,
		Role:      req.Role,
		Status:    models.StatusApplied,
		AppliedAt: appliedAt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := database.InsertApplication(h.DB, app); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save application"})
		return
	}

	// Publish Kafka event (fire-and-forget, don't block the response).
	go func() {
		event := models.ApplicationCreatedEvent{
			ApplicationID: app.ID,
			AppliedAt:     app.AppliedAt,
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := h.Producer.PublishApplicationCreated(ctx, event); err != nil {
			// Logged inside producer; non-blocking.
		}
	}()

	c.JSON(http.StatusCreated, app)
}

// UpdateApplication handles PATCH /applications/:id.
func (h *Handler) UpdateApplication(c *gin.Context) {
	id := c.Param("id")

	var req models.UpdateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !models.ValidStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status, must be one of: Applied, OA, Interview, Offer, Rejected"})
		return
	}

	if err := database.UpdateApplicationStatus(h.DB, id, req.Status); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	app, _ := database.GetApplicationByID(h.DB, id)
	c.JSON(http.StatusOK, app)
}

// ListApplications handles GET /applications.
func (h *Handler) ListApplications(c *gin.Context) {
	status := c.Query("status")

	apps, err := database.ListApplications(h.DB, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch applications"})
		return
	}

	if apps == nil {
		apps = []models.Application{}
	}

	c.JSON(http.StatusOK, apps)
}

// GetStats handles GET /applications/stats.
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := database.GetStats(h.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// UploadAttachment handles POST /applications/:id/attachment.
// Returns a pre-signed S3 URL for direct upload.
func (h *Handler) UploadAttachment(c *gin.Context) {
	id := c.Param("id")

	// Verify the application exists.
	_, err := database.GetApplicationByID(h.DB, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}

	url, objectKey, err := h.S3Client.GeneratePresignedUploadURL(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate upload URL"})
		return
	}

	// Store the S3 object key in the database.
	if err := database.UpdateS3ObjectKey(h.DB, id, objectKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update application"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"upload_url": url,
		"object_key": objectKey,
		"expires_in": "10 minutes",
		"method":     "PUT",
		"note":       "Upload your PDF using a PUT request to the upload_url",
	})
}
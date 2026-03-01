package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/SrushtiPatil01/Job-Tracker/internal/config"
	"github.com/SrushtiPatil01/Job-Tracker/internal/database"
	"github.com/SrushtiPatil01/Job-Tracker/internal/handlers"
	"github.com/SrushtiPatil01/Job-Tracker/internal/kafka"
	"github.com/SrushtiPatil01/Job-Tracker/internal/s3"
)

func main() {
	cfg := config.Load()

	// Connect to PostgreSQL.
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	// Initialize Kafka producer.
	producer := kafka.NewProducer(cfg.KafkaBroker)
	defer producer.Close()

	// Initialize S3 client.
	s3Client, err := s3.NewClient(cfg.AWSAccessKey, cfg.AWSSecretKey, cfg.AWSRegion, cfg.AWSS3Bucket)
	if err != nil {
		log.Fatalf("S3 client initialization failed: %v", err)
	}

	// Set up handler with dependencies.
	h := &handlers.Handler{
		DB:       db,
		Producer: producer,
		S3Client: s3Client,
	}

	// Set up Gin router.
	r := gin.Default()

	// Application routes.
	r.POST("/applications", h.CreateApplication)
	r.PATCH("/applications/:id", h.UpdateApplication)
	r.GET("/applications", h.ListApplications)
	r.GET("/applications/stats", h.GetStats)
	r.POST("/applications/:id/attachment", h.UploadAttachment)

	// Prometheus metrics endpoint.
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health check.
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	log.Printf("API server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/SrushtiPatil01/Job-Tracker/internal/config"
	"github.com/SrushtiPatil01/Job-Tracker/internal/database"
	"github.com/SrushtiPatil01/Job-Tracker/internal/kafka"
	"github.com/SrushtiPatil01/Job-Tracker/internal/models"
)

const seedCount = 10_000

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	producer := kafka.NewProducer(cfg.KafkaBroker)
	defer producer.Close()

	companies := []string{
		"Google", "Meta", "Amazon", "Apple", "Microsoft",
		"Netflix", "Stripe", "Databricks", "Airbnb", "Uber",
		"Coinbase", "Shopify", "Snowflake", "Palantir", "Figma",
		"Notion", "Vercel", "Supabase", "Linear", "Retool",
	}

	roles := []string{
		"Software Engineer", "Backend Engineer", "Full Stack Engineer",
		"Platform Engineer", "Infrastructure Engineer", "ML Engineer",
		"Data Engineer", "DevOps Engineer", "SRE", "Staff Engineer",
	}

	// Set applied_at to 8 days ago so all records trigger reminders.
	appliedAt := time.Now().Add(-8 * 24 * time.Hour)

	log.Printf("Seeding %d applications with applied_at = %s", seedCount, appliedAt.Format("2006-01-02"))
	start := time.Now()

	for i := 0; i < seedCount; i++ {
		id := uuid.New().String()
		company := companies[i%len(companies)]
		role := roles[i%len(roles)]

		app := &models.Application{
			ID:        id,
			Company:   fmt.Sprintf("%s-%d", company, i),
			Role:      role,
			Status:    models.StatusApplied,
			AppliedAt: appliedAt,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := database.InsertApplication(db, app); err != nil {
			log.Printf("Failed to insert application %d: %v", i, err)
			continue
		}

		// Publish Kafka event for each seeded application.
		event := models.ApplicationCreatedEvent{
			ApplicationID: id,
			AppliedAt:     appliedAt,
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := producer.PublishApplicationCreated(ctx, event); err != nil {
			log.Printf("Failed to publish event %d: %v", i, err)
		}
		cancel()

		if (i+1)%1000 == 0 {
			log.Printf("Seeded %d / %d applications", i+1, seedCount)
		}
	}

	elapsed := time.Since(start)
	log.Printf("Seeding complete: %d applications in %v", seedCount, elapsed)
	log.Println("Now run the consumer and observe Prometheus metrics for benchmark numbers.")
}
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/SrushtiPatil01/Job-Tracker/internal/config"
	"github.com/SrushtiPatil01/Job-Tracker/internal/database"
	"github.com/SrushtiPatil01/Job-Tracker/internal/email"
	"github.com/SrushtiPatil01/Job-Tracker/internal/kafka"
	redisclient "github.com/SrushtiPatil01/Job-Tracker/internal/redis"
)

func main() {
	cfg := config.Load()

	// Connect to PostgreSQL.
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	// Connect to Redis.
	rc, err := redisclient.Connect(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Redis connection failed: %v", err)
	}
	defer rc.Close()

	// Initialize email sender.
	sender := email.NewSender(cfg.SendGridAPIKey, cfg.SendGridFrom, cfg.ReminderToEmail)

	// Initialize Kafka consumer.
	consumer := kafka.NewConsumer(cfg.KafkaBroker, db, rc, sender)

	// Context that cancels on SIGINT/SIGTERM for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutdown signal received")
		cancel()
	}()

	// Expose Prometheus metrics on a separate port.
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		log.Println("Consumer metrics available on :9091/metrics")
		if err := http.ListenAndServe(":9091", mux); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	// On startup, run a batch check for any pending reminders
	// that may have been missed while the consumer was down.
	log.Println("Running initial batch reminder check...")
	if err := consumer.RunBatch(ctx); err != nil {
		log.Printf("Batch check error: %v", err)
	}

	// Start a periodic batch check every hour.
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				log.Println("Running periodic batch reminder check...")
				if err := consumer.RunBatch(ctx); err != nil {
					log.Printf("Periodic batch check error: %v", err)
				}
			}
		}
	}()

	// Start consuming Kafka events (blocks until context is cancelled).
	log.Println("Starting Kafka consumer loop...")
	if err := consumer.Run(ctx); err != nil {
		log.Fatalf("Consumer error: %v", err)
	}

	log.Println("Consumer shut down cleanly")
}
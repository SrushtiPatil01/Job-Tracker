package kafka

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"sync"
	"time"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/SrushtiPatil01/Job-Tracker/internal/database"
	"github.com/SrushtiPatil01/Job-Tracker/internal/email"
	"github.com/SrushtiPatil01/Job-Tracker/internal/metrics"
	"github.com/SrushtiPatil01/Job-Tracker/internal/models"
	redisclient "github.com/SrushtiPatil01/Job-Tracker/internal/redis"
)

// Consumer reads from Kafka and processes reminder events.
type Consumer struct {
	reader      *kafkago.Reader
	db          *sql.DB
	redisClient *redisclient.Client
	sender      *email.Sender
}

// NewConsumer creates a new Kafka consumer.
func NewConsumer(broker string, db *sql.DB, rc *redisclient.Client, sender *email.Sender) *Consumer {
	r := kafkago.NewReader(kafkago.ReaderConfig{
		Brokers:        []string{broker},
		Topic:          TopicApplicationsCreated,
		GroupID:        "reminder-consumer",
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
	})

	log.Println("Kafka consumer initialized")
	return &Consumer{
		reader:      r,
		db:          db,
		redisClient: rc,
		sender:      sender,
	}
}

// Run starts the consumer loop. It reads messages from Kafka, then for each
// message checks if a reminder should be sent. Processing of each application
// happens in its own goroutine for parallelism.
func (c *Consumer) Run(ctx context.Context) error {
	log.Println("Consumer started, listening for events...")

	for {
		select {
		case <-ctx.Done():
			log.Println("Consumer shutting down")
			return c.reader.Close()
		default:
		}

		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("Error reading Kafka message: %v", err)
			continue
		}

		// Track consumer lag.
		lag := time.Since(msg.Time).Seconds()
		metrics.KafkaConsumerLagSeconds.Set(lag)

		var event models.ApplicationCreatedEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("Failed to unmarshal event: %v", err)
			continue
		}

		go c.processEvent(ctx, event)
	}
}

// RunBatch processes all pending reminders from the database in parallel.
// This is used by the consumer on startup and periodically to catch up.
func (c *Consumer) RunBatch(ctx context.Context) error {
	threshold := time.Now().Add(-7 * 24 * time.Hour)
	apps, err := database.GetPendingReminders(c.db, threshold)
	if err != nil {
		return err
	}

	log.Printf("Found %d pending applications for reminders", len(apps))

	var wg sync.WaitGroup
	for _, app := range apps {
		wg.Add(1)
		go func(a models.Application) {
			defer wg.Done()
			c.processApplication(ctx, a)
		}(app)
	}

	wg.Wait()
	log.Println("Batch processing complete")
	return nil
}

func (c *Consumer) processEvent(ctx context.Context, event models.ApplicationCreatedEvent) {
	// Only send reminders if applied 7+ days ago.
	if time.Since(event.AppliedAt) < 7*24*time.Hour {
		return
	}

	app, err := database.GetApplicationByID(c.db, event.ApplicationID)
	if err != nil {
		log.Printf("Failed to fetch application %s: %v", event.ApplicationID, err)
		return
	}

	c.processApplication(ctx, *app)
}

func (c *Consumer) processApplication(ctx context.Context, app models.Application) {
	// Only remind if still in "Applied" status.
	if app.Status != models.StatusApplied {
		return
	}

	// Idempotency check: has a reminder already been sent this week?
	sent, err := c.redisClient.HasReminderBeenSent(ctx, app.ID)
	if err != nil {
		log.Printf("Redis check failed for %s: %v", app.ID, err)
		return
	}
	if sent {
		return
	}

	// Send the reminder email.
	if err := c.sender.SendReminder(app.Company, app.Role, app.ID); err != nil {
		log.Printf("Failed to send reminder for %s: %v", app.ID, err)
		metrics.RemindersFailedTotal.Inc()
		return
	}

	// Mark as sent in Redis.
	if err := c.redisClient.MarkReminderSent(ctx, app.ID); err != nil {
		log.Printf("Failed to set Redis key for %s: %v", app.ID, err)
	}

	metrics.RemindersSentTotal.Inc()
}

// Close shuts down the Kafka consumer.
func (c *Consumer) Close() error {
	return c.reader.Close()
}
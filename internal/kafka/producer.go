package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/SrushtiPatil01/Job-Tracker/internal/models"
)

const TopicApplicationsCreated = "applications.created"

// Producer publishes events to Kafka.
type Producer struct {
	writer *kafka.Writer
}

// NewProducer creates a Kafka producer for the applications.created topic.
func NewProducer(broker string) *Producer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(broker),
		Topic:        TopicApplicationsCreated,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
	}
	log.Printf("Kafka producer initialized for topic %s", TopicApplicationsCreated)
	return &Producer{writer: w}
}

// PublishApplicationCreated sends an ApplicationCreatedEvent to Kafka.
func (p *Producer) PublishApplicationCreated(ctx context.Context, event models.ApplicationCreatedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(event.ApplicationID),
		Value: data,
		Time:  time.Now(),
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		log.Printf("Failed to publish Kafka event: %v", err)
		return err
	}

	log.Printf("Published ApplicationCreated event for %s", event.ApplicationID)
	return nil
}

// Close shuts down the Kafka producer.
func (p *Producer) Close() error {
	return p.writer.Close()
}
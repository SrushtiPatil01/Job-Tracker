package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps the Redis client with idempotency helpers.
type Client struct {
	rdb *redis.Client
}

// Connect creates a new Redis client and verifies the connection.
func Connect(addr string) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	log.Println("Redis connected")
	return &Client{rdb: rdb}, nil
}

// reminderKey builds the idempotency key for a given application and ISO week.
// Format: reminder_sent:{applicationId}:{isoWeek}
func reminderKey(applicationID string, t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("reminder_sent:%s:%d-W%02d", applicationID, year, week)
}

// HasReminderBeenSent checks if a reminder was already sent for this
// application during the current ISO week.
func (c *Client) HasReminderBeenSent(ctx context.Context, applicationID string) (bool, error) {
	key := reminderKey(applicationID, time.Now())
	val, err := c.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return val > 0, nil
}

// MarkReminderSent sets the idempotency key with a 7-day TTL.
func (c *Client) MarkReminderSent(ctx context.Context, applicationID string) error {
	key := reminderKey(applicationID, time.Now())
	return c.rdb.Set(ctx, key, "1", 7*24*time.Hour).Err()
}

// Close shuts down the Redis connection.
func (c *Client) Close() error {
	return c.rdb.Close()
}
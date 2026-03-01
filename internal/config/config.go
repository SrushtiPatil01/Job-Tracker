package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL     string
	KafkaBroker     string
	RedisURL        string
	SendGridAPIKey  string
	SendGridFrom    string
	ReminderToEmail string
	AWSAccessKey    string
	AWSSecretKey    string
	AWSRegion       string
	AWSS3Bucket     string
	Port            string
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, reading from environment")
	}

	return &Config{
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/jobtracker?sslmode=disable"),
		KafkaBroker:     getEnv("KAFKA_BROKER", "localhost:9092"),
		RedisURL:        getEnv("REDIS_URL", "localhost:6379"),
		SendGridAPIKey:  getEnv("SENDGRID_API_KEY", ""),
		SendGridFrom:    getEnv("SENDGRID_FROM_EMAIL", "noreply@jobtracker.dev"),
		ReminderToEmail: getEnv("REMINDER_TO_EMAIL", ""),
		AWSAccessKey:    getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretKey:    getEnv("AWS_SECRET_ACCESS_KEY", ""),
		AWSRegion:       getEnv("AWS_REGION", "us-east-1"),
		AWSS3Bucket:     getEnv("AWS_S3_BUCKET", "job-tracker-attachments"),
		Port:            getEnv("PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
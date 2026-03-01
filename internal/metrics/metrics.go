package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// RemindersSentTotal tracks how many reminder emails were successfully sent.
	RemindersSentTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "reminders_sent_total",
		Help: "Total number of reminder emails successfully sent",
	})

	// RemindersFailedTotal tracks how many reminder emails failed to send.
	RemindersFailedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "reminders_failed_total",
		Help: "Total number of reminder emails that failed to send",
	})

	// KafkaConsumerLagSeconds tracks how far behind the consumer is.
	KafkaConsumerLagSeconds = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "kafka_consumer_lag_seconds",
		Help: "Seconds of lag between event timestamp and consumer processing time",
	})
)
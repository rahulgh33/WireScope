package queue

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Prometheus metrics for queue monitoring
// Requirement: 6.2 - Queue lag monitoring
var (
	queueLagMessages = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "queue_lag_messages",
			Help: "Number of messages pending in the queue (not yet delivered to consumers)",
		},
	)

	queueAckPendingMessages = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "queue_ack_pending_messages",
			Help: "Number of messages delivered to consumers but not yet acknowledged",
		},
	)

	dlqMessagesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "dlq_messages_total",
			Help: "Total number of messages sent to the dead letter queue",
		},
	)

	natsReconnectsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "nats_reconnects_total",
			Help: "Total number of NATS reconnection events",
		},
	)

	metricsOnce sync.Once
)

func init() {
	metricsOnce.Do(func() {
		prometheus.DefaultRegisterer.MustRegister(queueLagMessages)
		prometheus.DefaultRegisterer.MustRegister(queueAckPendingMessages)
		prometheus.DefaultRegisterer.MustRegister(dlqMessagesTotal)
		prometheus.DefaultRegisterer.MustRegister(natsReconnectsTotal)
	})
}

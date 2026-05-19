package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	messagesSent     *prometheus.CounterVec
	messagesReceived prometheus.Counter
	messagesFailed   prometheus.Counter
	rateLimitHits    prometheus.Counter
	requestDuration  *prometheus.HistogramVec
	workerActive     prometheus.Gauge
	queueDepth       *prometheus.GaugeVec
}

func New() *Metrics {
	m := &Metrics{
		messagesSent: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "wa_messages_sent_total",
			Help: "Total outbound messages sent",
		}, []string{"status", "channel"}),

		messagesReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wa_messages_received_total",
			Help: "Total inbound messages received",
		}),

		messagesFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wa_messages_failed_total",
			Help: "Total messages that failed to send",
		}),

		rateLimitHits: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wa_rate_limit_hits_total",
			Help: "Total times the WABA rate limiter was hit",
		}),

		requestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "wa_http_request_duration_seconds",
			Help:    "HTTP request duration to Meta API",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path", "status"}),

		workerActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "wa_worker_active",
			Help: "Number of active workers in the pool",
		}),

		queueDepth: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "wa_queue_depth",
			Help: "Current queue depth",
		}, []string{"queue"}),
	}

	prometheus.MustRegister(
		m.messagesSent,
		m.messagesReceived,
		m.messagesFailed,
		m.rateLimitHits,
		m.requestDuration,
		m.workerActive,
		m.queueDepth,
	)

	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.Handler()
}

func (m *Metrics) IncMessagesSent(status, channel string) {
	m.messagesSent.WithLabelValues(status, channel).Inc()
}

func (m *Metrics) IncMessagesReceived() {
	m.messagesReceived.Inc()
}

func (m *Metrics) IncMessagesFailed() {
	m.messagesFailed.Inc()
}

func (m *Metrics) IncRateLimitHits() {
	m.rateLimitHits.Inc()
}

func (m *Metrics) ObserveRequest(start time.Time, method, path string, status int) {
	m.requestDuration.WithLabelValues(method, path, http.StatusText(status)).Observe(time.Since(start).Seconds())
}

func (m *Metrics) ObserveMetaRequest(start time.Time, method, path string, status int) {
	m.ObserveRequest(start, method, path, status)
}

func (m *Metrics) SetWorkerActive(n int) {
	m.workerActive.Set(float64(n))
}

func (m *Metrics) SetQueueDepth(queue string, depth int) {
	m.queueDepth.WithLabelValues(queue).Set(float64(depth))
}

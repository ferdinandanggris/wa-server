package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type TokenBucket struct {
	rate     float64
	burst    int
	tokens   float64
	lastTime time.Time
	mu       sync.Mutex
}

func NewTokenBucket(rate float64, burst int) *TokenBucket {
	return &TokenBucket{
		rate:     rate,
		burst:    burst,
		tokens:   float64(burst),
		lastTime: time.Now(),
	}
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastTime).Seconds()
	tb.lastTime = now
	tb.tokens += elapsed * tb.rate
	if tb.tokens > float64(tb.burst) {
		tb.tokens = float64(tb.burst)
	}

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

func (tb *TokenBucket) Wait(ctx context.Context) error {
	for {
		if tb.Allow() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second / time.Duration(tb.rate)):
		}
	}
}

type MetricsRecorder interface {
	IncRateLimitHits()
	ObserveRequest(start time.Time, method, path string, status int)
}

type Transport struct {
	inner   http.RoundTripper
	bucket  *TokenBucket
	metrics MetricsRecorder
}

func NewTransport(inner http.RoundTripper, bucket *TokenBucket, metrics MetricsRecorder) *Transport {
	if inner == nil {
		inner = http.DefaultTransport
	}
	return &Transport{inner: inner, bucket: bucket, metrics: metrics}
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	if !t.bucket.Allow() {
		t.metrics.IncRateLimitHits()
		if err := t.bucket.Wait(req.Context()); err != nil {
			return nil, fmt.Errorf("rate limit wait: %w", err)
		}
	}

	resp, err := t.inner.RoundTrip(req)
	status := http.StatusInternalServerError
	if resp != nil {
		status = resp.StatusCode
	}
	t.metrics.ObserveRequest(start, req.Method, req.URL.Path, status)
	return resp, err
}

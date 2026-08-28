package cache

import (
	"context"
	"sync"
	"time"
)

// Metrics tracks cache performance metrics.
type Metrics struct {
	mu            sync.RWMutex
	hits          int64
	misses        int64
	totalRequests int64
	totalLatency  time.Duration
	errors        int64
}

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{}
}

// RecordHit records a cache hit.
func (m *Metrics) RecordHit(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hits++
	m.totalRequests++
	m.totalLatency += latency
}

// RecordMiss records a cache miss.
func (m *Metrics) RecordMiss(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.misses++
	m.totalRequests++
	m.totalLatency += latency
}

// RecordError records a cache error.
func (m *Metrics) RecordError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors++
}

// GetHitRate returns the cache hit rate as a percentage.
func (m *Metrics) GetHitRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.totalRequests == 0 {
		return 0.0
	}

	return float64(m.hits) / float64(m.totalRequests) * 100.0
}

// GetAverageLatency returns the average latency for cache operations.
func (m *Metrics) GetAverageLatency() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.totalRequests == 0 {
		return 0
	}

	return m.totalLatency / time.Duration(m.totalRequests)
}

// GetHits returns the number of cache hits.
func (m *Metrics) GetHits() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.hits
}

// GetMisses returns the number of cache misses.
func (m *Metrics) GetMisses() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.misses
}

// GetTotalRequests returns the total number of cache requests.
func (m *Metrics) GetTotalRequests() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.totalRequests
}

// GetErrors returns the number of cache errors.
func (m *Metrics) GetErrors() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.errors
}

// Reset resets all metrics to zero.
func (m *Metrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hits = 0
	m.misses = 0
	m.totalRequests = 0
	m.totalLatency = 0
	m.errors = 0
}

// MetricsWrapper wraps a Strategy with metrics collection.
type MetricsWrapper struct {
	strategy Strategy
	metrics  *Metrics
}

// NewMetricsWrapper creates a new MetricsWrapper.
func NewMetricsWrapper(strategy Strategy) *MetricsWrapper {
	return &MetricsWrapper{
		strategy: strategy,
		metrics:  NewMetrics(),
	}
}

// Get retrieves a value from cache by key with metrics tracking.
func (w *MetricsWrapper) Get(ctx context.Context, key string) ([]byte, error) {
	start := time.Now()
	data, err := w.strategy.Get(ctx, key)
	latency := time.Since(start)

	if err != nil {
		w.metrics.RecordError()
		return nil, err
	}

	if data != nil {
		w.metrics.RecordHit(latency)
	} else {
		w.metrics.RecordMiss(latency)
	}

	return data, nil
}

// Set stores a value in cache with metrics tracking.
func (w *MetricsWrapper) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	start := time.Now()
	err := w.strategy.Set(ctx, key, data, ttl)
	latency := time.Since(start)

	if err != nil {
		w.metrics.RecordError()
	} else {
		w.metrics.RecordMiss(latency) // Set is considered a "miss" for metrics
	}

	return err
}

// Invalidate removes a value from cache.
func (w *MetricsWrapper) Invalidate(ctx context.Context, key string) error {
	return w.strategy.Invalidate(ctx, key)
}

// GetWithFallback retrieves a value with fallback and metrics tracking.
func (w *MetricsWrapper) GetWithFallback(ctx context.Context, key string, ttl time.Duration, fallback func() ([]byte, error)) ([]byte, error) {
	start := time.Now()
	data, err := w.strategy.GetWithFallback(ctx, key, ttl, fallback)
	latency := time.Since(start)

	if err != nil {
		w.metrics.RecordError()
		return nil, err
	}

	// Try to determine if it was a hit or miss by checking cache first
	_, cacheErr := w.strategy.Get(ctx, key)
	if cacheErr == nil {
		w.metrics.RecordHit(latency)
	} else {
		w.metrics.RecordMiss(latency)
	}

	return data, nil
}

// WarmUp pre-warms the cache.
func (w *MetricsWrapper) WarmUp(ctx context.Context, keys []string, generator func(string) ([]byte, error)) error {
	return w.strategy.WarmUp(ctx, keys, generator)
}

// GetMetrics returns the metrics instance.
func (w *MetricsWrapper) GetMetrics() *Metrics {
	return w.metrics
}

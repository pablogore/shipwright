package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCacheMetrics(t *testing.T) {
	metrics := NewCacheMetrics()

	assert.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics.GetHits())
	assert.Equal(t, int64(0), metrics.GetMisses())
	assert.Equal(t, int64(0), metrics.GetTotalRequests())
}

func TestCacheMetrics_RecordHit(t *testing.T) {
	metrics := NewCacheMetrics()

	metrics.RecordHit(10 * time.Millisecond)

	assert.Equal(t, int64(1), metrics.GetHits())
	assert.Equal(t, int64(1), metrics.GetTotalRequests())
}

func TestCacheMetrics_RecordMiss(t *testing.T) {
	metrics := NewCacheMetrics()

	metrics.RecordMiss(20 * time.Millisecond)

	assert.Equal(t, int64(1), metrics.GetMisses())
	assert.Equal(t, int64(1), metrics.GetTotalRequests())
}

func TestCacheMetrics_GetHitRate(t *testing.T) {
	metrics := NewCacheMetrics()

	// No requests
	assert.Equal(t, 0.0, metrics.GetHitRate())

	// 2 hits, 1 miss
	metrics.RecordHit(10 * time.Millisecond)
	metrics.RecordHit(10 * time.Millisecond)
	metrics.RecordMiss(20 * time.Millisecond)

	hitRate := metrics.GetHitRate()
	assert.InDelta(t, 66.67, hitRate, 0.1) // 2/3 * 100
}

func TestCacheMetrics_GetAverageLatency(t *testing.T) {
	metrics := NewCacheMetrics()

	metrics.RecordHit(10 * time.Millisecond)
	metrics.RecordHit(20 * time.Millisecond)
	metrics.RecordMiss(30 * time.Millisecond)

	avgLatency := metrics.GetAverageLatency()
	expectedLatency := 20 * time.Millisecond
	tolerance := 1 * time.Millisecond
	assert.True(t, avgLatency >= expectedLatency-tolerance && avgLatency <= expectedLatency+tolerance,
		"expected latency around %v, got %v", expectedLatency, avgLatency)
}

func TestCacheMetrics_Reset(t *testing.T) {
	metrics := NewCacheMetrics()

	metrics.RecordHit(10 * time.Millisecond)
	metrics.RecordMiss(20 * time.Millisecond)
	metrics.RecordError()

	metrics.Reset()

	assert.Equal(t, int64(0), metrics.GetHits())
	assert.Equal(t, int64(0), metrics.GetMisses())
	assert.Equal(t, int64(0), metrics.GetTotalRequests())
	assert.Equal(t, int64(0), metrics.GetErrors())
}

func TestNewMetricsWrapper(t *testing.T) {
	mockStrategy := NewMockStrategy()
	wrapper := NewMetricsWrapper(mockStrategy)

	assert.NotNil(t, wrapper)
	assert.Equal(t, mockStrategy, wrapper.strategy)
	assert.NotNil(t, wrapper.metrics)
}

func TestMetricsWrapper_GetMetrics(t *testing.T) {
	wrapper := NewMetricsWrapper(NewMockStrategy())

	metrics := wrapper.GetMetrics()

	assert.NotNil(t, metrics)
}

func TestMetricsWrapper_Get_TracksMetrics(t *testing.T) {
	mockStrategy := NewMockStrategy()
	mockStrategy.GetFunc = func(ctx context.Context, key string) ([]byte, error) {
		return []byte("data"), nil
	}

	wrapper := NewMetricsWrapper(mockStrategy)
	ctx := context.Background()

	data, err := wrapper.Get(ctx, "test-key")

	require.NoError(t, err)
	assert.Equal(t, []byte("data"), data)
	assert.Equal(t, int64(1), wrapper.GetMetrics().GetHits())
	assert.Equal(t, int64(1), wrapper.GetMetrics().GetTotalRequests())
}


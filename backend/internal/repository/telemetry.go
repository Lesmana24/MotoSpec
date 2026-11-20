package repository

import (
	"math"
	"sync/atomic"
)

// TelemetryStore maintains thread-safe atomic counters for system telemetry.
type TelemetryStore struct {
	cacheHits          atomic.Uint64
	cacheMisses        atomic.Uint64
	llmPrimarySuccess  atomic.Uint64
	llmFallbackSuccess atomic.Uint64
	llmTotalFailures   atomic.Uint64
}

// TelemetrySnapshot represents an immutable point-in-time view of telemetry counters.
type TelemetrySnapshot struct {
	CacheHits          uint64  `json:"cache_hits"`
	CacheMisses        uint64  `json:"cache_misses"`
	CacheHitRatioPct   float64 `json:"cache_hit_ratio_pct"`
	LLMPrimarySuccess  uint64  `json:"llm_primary_success"`
	LLMFallbackSuccess uint64  `json:"llm_fallback_success"`
	LLMTotalFailures   uint64  `json:"llm_total_failures"`
}

// NewTelemetryStore initializes a new TelemetryStore.
func NewTelemetryStore() *TelemetryStore {
	return &TelemetryStore{}
}

// IncCacheHit atomically increments cache hit counter.
func (t *TelemetryStore) IncCacheHit() uint64 {
	return t.cacheHits.Add(1)
}

// IncCacheMiss atomically increments cache miss counter.
func (t *TelemetryStore) IncCacheMiss() uint64 {
	return t.cacheMisses.Add(1)
}

// IncLLMPrimarySuccess atomically increments primary LLM success counter.
func (t *TelemetryStore) IncLLMPrimarySuccess() uint64 {
	return t.llmPrimarySuccess.Add(1)
}

// IncLLMFallbackSuccess atomically increments fallback LLM success counter.
func (t *TelemetryStore) IncLLMFallbackSuccess() uint64 {
	return t.llmFallbackSuccess.Add(1)
}

// IncLLMTotalFailure atomically increments total LLM failure counter.
func (t *TelemetryStore) IncLLMTotalFailure() uint64 {
	return t.llmTotalFailures.Add(1)
}

// Snapshot returns the current counters along with calculated hit ratio.
func (t *TelemetryStore) Snapshot() TelemetrySnapshot {
	hits := t.cacheHits.Load()
	misses := t.cacheMisses.Load()
	total := hits + misses

	var ratio float64
	if total > 0 {
		ratio = (float64(hits) / float64(total)) * 100.0
		ratio = math.Round(ratio*10) / 10
	}

	return TelemetrySnapshot{
		CacheHits:          hits,
		CacheMisses:        misses,
		CacheHitRatioPct:   ratio,
		LLMPrimarySuccess:  t.llmPrimarySuccess.Load(),
		LLMFallbackSuccess: t.llmFallbackSuccess.Load(),
		LLMTotalFailures:   t.llmTotalFailures.Load(),
	}
}

// Reset atomically resets all counters to 0 (useful for testing).
func (t *TelemetryStore) Reset() {
	t.cacheHits.Store(0)
	t.cacheMisses.Store(0)
	t.llmPrimarySuccess.Store(0)
	t.llmFallbackSuccess.Store(0)
	t.llmTotalFailures.Store(0)
}

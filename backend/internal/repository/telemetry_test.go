package repository_test

import (
	"sync"
	"testing"

	"github.com/motospec/backend/internal/repository"
)

func TestTelemetryStore_InitialState(t *testing.T) {
	store := repository.NewTelemetryStore()
	snap := store.Snapshot()

	if snap.CacheHits != 0 || snap.CacheMisses != 0 || snap.CacheHitRatioPct != 0.0 {
		t.Fatalf("expected all zeros on initial state, got %+v", snap)
	}
	if snap.LLMPrimarySuccess != 0 || snap.LLMFallbackSuccess != 0 || snap.LLMTotalFailures != 0 {
		t.Fatalf("expected zero LLM counters, got %+v", snap)
	}
}

func TestTelemetryStore_HitRatioCalculation(t *testing.T) {
	store := repository.NewTelemetryStore()

	// 8 hits, 2 misses => 80.0%
	for i := 0; i < 8; i++ {
		store.IncCacheHit()
	}
	for i := 0; i < 2; i++ {
		store.IncCacheMiss()
	}

	snap := store.Snapshot()
	if snap.CacheHits != 8 {
		t.Errorf("expected 8 hits, got %d", snap.CacheHits)
	}
	if snap.CacheMisses != 2 {
		t.Errorf("expected 2 misses, got %d", snap.CacheMisses)
	}
	if snap.CacheHitRatioPct != 80.0 {
		t.Errorf("expected 80.0%% hit ratio, got %.2f", snap.CacheHitRatioPct)
	}
}

func TestTelemetryStore_ConcurrentIncrements_RaceSafe(t *testing.T) {
	store := repository.NewTelemetryStore()
	var wg sync.WaitGroup

	goroutines := 50
	incrementsPerGoroutine := 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < incrementsPerGoroutine; j++ {
				store.IncCacheHit()
				store.IncCacheMiss()
				store.IncLLMPrimarySuccess()
				store.IncLLMFallbackSuccess()
				store.IncLLMTotalFailure()
			}
		}()
	}

	wg.Wait()

	snap := store.Snapshot()
	expected := uint64(goroutines * incrementsPerGoroutine)

	if snap.CacheHits != expected {
		t.Errorf("expected %d CacheHits, got %d", expected, snap.CacheHits)
	}
	if snap.CacheMisses != expected {
		t.Errorf("expected %d CacheMisses, got %d", expected, snap.CacheMisses)
	}
	if snap.LLMPrimarySuccess != expected {
		t.Errorf("expected %d LLMPrimarySuccess, got %d", expected, snap.LLMPrimarySuccess)
	}
	if snap.LLMFallbackSuccess != expected {
		t.Errorf("expected %d LLMFallbackSuccess, got %d", expected, snap.LLMFallbackSuccess)
	}
	if snap.LLMTotalFailures != expected {
		t.Errorf("expected %d LLMTotalFailures, got %d", expected, snap.LLMTotalFailures)
	}
	if snap.CacheHitRatioPct != 50.0 {
		t.Errorf("expected 50.0%% hit ratio, got %.2f", snap.CacheHitRatioPct)
	}
}

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/motospec/backend/internal/repository"
)

func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *repository.RedisSpecCache) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cache := repository.NewRedisSpecCache(client, 24*time.Hour)
	return mr, cache
}

func sampleSpec() *repository.BikeSpecDTO {
	modelID := uuid.New()
	compID := uuid.New()

	return &repository.BikeSpecDTO{
		ID:             modelID,
		CanonicalSlug:  "honda-vario-150",
		Brand:          "Honda",
		DisplayName:    "Honda Vario 150",
		Category:       "matic",
		EngineCC:       150,
		IsVerified:     true,
		SourceProvider: "llm_primary",
		CreatedAt:      time.Now().Truncate(time.Second),
		Components: []repository.SpecComponentDTO{
			{
				ID:             compID,
				BikeModelID:    modelID,
				Code:           "engine_oil",
				Name:           "Oli Mesin",
				Category:       "lubrication",
				IntervalKM:     4000,
				IntervalMonths: 4,
				UrgencyLevel:   "high",
				Notes:          "Gunakan oli 10W-30",
			},
		},
	}
}

func TestRedisSpecCache_GetSpec_Miss(t *testing.T) {
	mr, cache := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	spec, err := cache.GetSpec(ctx, "non-existent-slug")
	if err == nil {
		t.Fatalf("expected error on cache miss, got nil")
	}
	if err != repository.ErrCacheMiss {
		t.Fatalf("expected ErrCacheMiss, got: %v", err)
	}
	if spec != nil {
		t.Fatalf("expected nil spec, got: %+v", spec)
	}
}

func TestRedisSpecCache_SetAndGetSpec_Success(t *testing.T) {
	mr, cache := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	original := sampleSpec()

	if err := cache.SetSpec(ctx, original); err != nil {
		t.Fatalf("SetSpec failed: %v", err)
	}

	retrieved, err := cache.GetSpec(ctx, original.CanonicalSlug)
	if err != nil {
		t.Fatalf("GetSpec failed: %v", err)
	}

	if retrieved.ID != original.ID {
		t.Errorf("expected ID %v, got %v", original.ID, retrieved.ID)
	}
	if retrieved.CanonicalSlug != original.CanonicalSlug {
		t.Errorf("expected CanonicalSlug %v, got %v", original.CanonicalSlug, retrieved.CanonicalSlug)
	}
	if len(retrieved.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(retrieved.Components))
	}
	if retrieved.Components[0].Code != "engine_oil" {
		t.Errorf("expected component code 'engine_oil', got '%s'", retrieved.Components[0].Code)
	}
}

func TestRedisSpecCache_TTL_Expiration(t *testing.T) {
	mr, cache := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	original := sampleSpec()

	if err := cache.SetSpec(ctx, original, 10*time.Second); err != nil {
		t.Fatalf("SetSpec failed: %v", err)
	}

	// Advance time past TTL
	mr.FastForward(11 * time.Second)

	_, err := cache.GetSpec(ctx, original.CanonicalSlug)
	if err != repository.ErrCacheMiss {
		t.Fatalf("expected ErrCacheMiss after expiration, got %v", err)
	}
}

func TestRedisSpecCache_DeleteSpec(t *testing.T) {
	mr, cache := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	original := sampleSpec()

	if err := cache.SetSpec(ctx, original); err != nil {
		t.Fatalf("SetSpec failed: %v", err)
	}

	if err := cache.DeleteSpec(ctx, original.CanonicalSlug); err != nil {
		t.Fatalf("DeleteSpec failed: %v", err)
	}

	_, err := cache.GetSpec(ctx, original.CanonicalSlug)
	if err != repository.ErrCacheMiss {
		t.Fatalf("expected ErrCacheMiss after delete, got %v", err)
	}
}

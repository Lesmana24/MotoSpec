package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	// ErrCacheMiss indicates the requested key is not present in cache.
	ErrCacheMiss = errors.New("repository: cache miss")
)

const (
	defaultSpecPrefix = "bike:spec:"
	DefaultSpecTTL    = 24 * time.Hour
)

// SpecCache defines the interface for motorcycle spec caching.
type SpecCache interface {
	GetSpec(ctx context.Context, canonicalSlug string) (*BikeSpecDTO, error)
	SetSpec(ctx context.Context, spec *BikeSpecDTO, ttl ...time.Duration) error
	DeleteSpec(ctx context.Context, canonicalSlug string) error
}

// RedisSpecCache implements SpecCache backed by Redis.
type RedisSpecCache struct {
	client     redis.Cmdable
	keyPrefix  string
	defaultTTL time.Duration
}

// NewRedisSpecCache creates a new RedisSpecCache instance.
func NewRedisSpecCache(client redis.Cmdable, defaultTTL time.Duration) *RedisSpecCache {
	if defaultTTL <= 0 {
		defaultTTL = DefaultSpecTTL
	}
	return &RedisSpecCache{
		client:     client,
		keyPrefix:  defaultSpecPrefix,
		defaultTTL: defaultTTL,
	}
}

func (c *RedisSpecCache) formatKey(slug string) string {
	return c.keyPrefix + slug
}

// GetSpec retrieves a cached motorcycle specification by canonical slug.
func (c *RedisSpecCache) GetSpec(ctx context.Context, canonicalSlug string) (*BikeSpecDTO, error) {
	key := c.formatKey(canonicalSlug)
	payload, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrCacheMiss
		}
		return nil, fmt.Errorf("failed to get spec from redis: %w", err)
	}

	var spec BikeSpecDTO
	if err := json.Unmarshal(payload, &spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached spec: %w", err)
	}

	return &spec, nil
}

// SetSpec stores a motorcycle specification in Redis with a TTL.
func (c *RedisSpecCache) SetSpec(ctx context.Context, spec *BikeSpecDTO, ttl ...time.Duration) error {
	if spec == nil {
		return errors.New("cannot cache nil spec")
	}

	payload, err := json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("failed to marshal spec for caching: %w", err)
	}

	duration := c.defaultTTL
	if len(ttl) > 0 && ttl[0] > 0 {
		duration = ttl[0]
	}

	key := c.formatKey(spec.CanonicalSlug)
	if err := c.client.Set(ctx, key, payload, duration).Err(); err != nil {
		return fmt.Errorf("failed to set spec in redis: %w", err)
	}

	return nil
}

// DeleteSpec invalidates a cached motorcycle specification.
func (c *RedisSpecCache) DeleteSpec(ctx context.Context, canonicalSlug string) error {
	key := c.formatKey(canonicalSlug)
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete spec from redis: %w", err)
	}
	return nil
}

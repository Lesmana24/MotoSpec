package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/motospec/backend/internal/llm"
	"github.com/motospec/backend/internal/repository"
	"github.com/motospec/backend/internal/repository/db"
)

var (
	ErrInvalidQuery = errors.New("service: query must be at least 2 characters")
	ErrNotFound     = errors.New("service: resource not found")
)

// SpecMeta contains telemetry metadata for spec responses.
type SpecMeta struct {
	Source            string  `json:"source"`
	ProviderUsed      *string `json:"provider_used"`
	FallbackTriggered bool    `json:"fallback_triggered"`
	FallbackReason    string  `json:"fallback_reason,omitempty"`
	LatencyMS         int64   `json:"latency_ms"`
	CanonicalSlug     string  `json:"canonical_slug"`
}

// SpecSearchResult wraps the spec payload with search metadata.
type SpecSearchResult struct {
	Meta SpecMeta                `json:"meta"`
	Data *repository.BikeSpecDTO `json:"data"`
}

// SpecListMeta contains metadata for the seeded bike models list.
type SpecListMeta struct {
	Source    string `json:"source"`
	LatencyMS int64  `json:"latency_ms"`
}

// SpecListResult wraps the list of seeded bike models.
type SpecListResult struct {
	Meta SpecListMeta   `json:"meta"`
	Data []db.BikeModel `json:"data"`
}

// SpecService orchestrates spec lookup across Redis, PostgreSQL, and LLM Orchestrator.
type SpecService struct {
	store        repository.Store
	cache        repository.SpecCache
	orchestrator *llm.Orchestrator
	telemetry    *repository.TelemetryStore
	sfGroup      singleflight.Group
}

// NewSpecService creates a new SpecService instance.
func NewSpecService(
	store repository.Store,
	cache repository.SpecCache,
	orchestrator *llm.Orchestrator,
	telemetry *repository.TelemetryStore,
) *SpecService {
	return &SpecService{
		store:        store,
		cache:        cache,
		orchestrator: orchestrator,
		telemetry:    telemetry,
	}
}

type generationEnvelope struct {
	DTO               *repository.BikeSpecDTO
	Source            string
	ProviderUsed      *string
	FallbackTriggered bool
	FallbackReason    string
}

// SearchAndSeed resolves a motorcycle query via Redis -> PostgreSQL -> LLM Orchestrator.
func (s *SpecService) SearchAndSeed(ctx context.Context, rawQuery string) (*SpecSearchResult, error) {
	rawQuery = strings.TrimSpace(rawQuery)
	if len(rawQuery) < 2 {
		return nil, ErrInvalidQuery
	}

	canonicalSlug := NormalizeQuery(rawQuery)
	if canonicalSlug == "" {
		return nil, ErrInvalidQuery
	}

	startTime := time.Now()

	// 1. Redis Lookup
	cached, err := s.cache.GetSpec(ctx, canonicalSlug)
	if err == nil && cached != nil {
		s.telemetry.IncCacheHit()
		return &SpecSearchResult{
			Meta: SpecMeta{
				Source:            "redis_cache",
				ProviderUsed:      nil,
				FallbackTriggered: false,
				LatencyMS:         time.Since(startTime).Milliseconds(),
				CanonicalSlug:     canonicalSlug,
			},
			Data: cached,
		}, nil
	}

	s.telemetry.IncCacheMiss()

	// 2. PostgreSQL Lookup
	model, err := s.store.GetBikeModelBySlug(ctx, canonicalSlug)
	if err == nil {
		comps, compErr := s.store.ListSpecComponentsByModelID(ctx, model.ID)
		if compErr == nil {
			dto := assembleSpecDTO(model, comps)
			// Backfill Redis (24h TTL)
			if cacheErr := s.cache.SetSpec(ctx, dto); cacheErr != nil {
				slog.Warn("failed to backfill spec into redis", "slug", canonicalSlug, "error", cacheErr)
			}

			return &SpecSearchResult{
				Meta: SpecMeta{
					Source:            "postgres_db",
					ProviderUsed:      nil,
					FallbackTriggered: false,
					LatencyMS:         time.Since(startTime).Milliseconds(),
					CanonicalSlug:     canonicalSlug,
				},
				Data: dto,
			}, nil
		}
	}

	// 3. Uncached: Protect concurrent LLM seeding using singleflight
	sfKey := canonicalSlug
	v, sfErr, _ := s.sfGroup.Do(sfKey, func() (interface{}, error) {
		// Double-check DB in case another goroutine already completed seeding
		if existingModel, err := s.store.GetBikeModelBySlug(ctx, canonicalSlug); err == nil {
			if existingComps, err := s.store.ListSpecComponentsByModelID(ctx, existingModel.ID); err == nil {
				dto := assembleSpecDTO(existingModel, existingComps)
				_ = s.cache.SetSpec(ctx, dto)
				return &generationEnvelope{
					DTO:               dto,
					Source:            "postgres_db",
					ProviderUsed:      nil,
					FallbackTriggered: false,
				}, nil
			}
		}

		// Invoke LLM Orchestrator
		genRes, err := s.orchestrator.GenerateSpec(ctx, canonicalSlug, rawQuery)
		if err != nil {
			return nil, err
		}

		// Persist into PostgreSQL within an atomic transaction
		var createdModel db.BikeModel
		var createdComps []db.BikeSpecComponent

		txErr := s.store.ExecTx(ctx, func(q *db.Queries) error {
			m, err := q.CreateBikeModel(ctx, db.CreateBikeModelParams{
				CanonicalSlug:  canonicalSlug,
				Brand:          genRes.Spec.Brand,
				DisplayName:    genRes.Spec.DisplayName,
				Category:       genRes.Spec.Category,
				EngineCc:       int32(genRes.Spec.EngineCC),
				IsVerified:     false,
				SourceProvider: genRes.ProviderUsed,
			})
			if err != nil {
				return fmt.Errorf("failed to insert bike_models: %w", err)
			}
			createdModel = m

			createdComps = make([]db.BikeSpecComponent, len(genRes.Spec.Components))
			for i, comp := range genRes.Spec.Components {
				c, err := q.CreateSpecComponent(ctx, db.CreateSpecComponentParams{
					BikeModelID:    m.ID,
					Code:           comp.Code,
					Name:           comp.Name,
					Category:       comp.Category,
					IntervalKm:     int32(comp.IntervalKM),
					IntervalMonths: int32(comp.IntervalMonths),
					UrgencyLevel:   comp.UrgencyLevel,
					Notes:          comp.Notes,
				})
				if err != nil {
					return fmt.Errorf("failed to insert bike_spec_components: %w", err)
				}
				createdComps[i] = c
			}
			return nil
		})

		if txErr != nil {
			return nil, fmt.Errorf("failed to persist spec into database: %w", txErr)
		}

		dto := assembleSpecDTO(createdModel, createdComps)

		// Backfill Redis
		if err := s.cache.SetSpec(ctx, dto); err != nil {
			slog.Warn("failed to backfill spec into redis", "slug", canonicalSlug, "error", err)
		}

		source := "llm_primary"
		if genRes.FallbackTriggered {
			source = "llm_fallback"
		}

		return &generationEnvelope{
			DTO:               dto,
			Source:            source,
			ProviderUsed:      &genRes.ProviderUsed,
			FallbackTriggered: genRes.FallbackTriggered,
			FallbackReason:    genRes.FallbackReason,
		}, nil
	})

	if sfErr != nil {
		return nil, sfErr
	}

	envelope := v.(*generationEnvelope)
	return &SpecSearchResult{
		Meta: SpecMeta{
			Source:            envelope.Source,
			ProviderUsed:      envelope.ProviderUsed,
			FallbackTriggered: envelope.FallbackTriggered,
			FallbackReason:    envelope.FallbackReason,
			LatencyMS:         time.Since(startTime).Milliseconds(),
			CanonicalSlug:     canonicalSlug,
		},
		Data: envelope.DTO,
	}, nil
}

// ListSeededModels returns all motorcycles currently persisted in PostgreSQL.
func (s *SpecService) ListSeededModels(ctx context.Context) (*SpecListResult, error) {
	startTime := time.Now()
	models, err := s.store.ListBikeModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list bike models: %w", err)
	}

	return &SpecListResult{
		Meta: SpecListMeta{
			Source:    "postgres_db",
			LatencyMS: time.Since(startTime).Milliseconds(),
		},
		Data: models,
	}, nil
}

func assembleSpecDTO(model db.BikeModel, comps []db.BikeSpecComponent) *repository.BikeSpecDTO {
	dto := &repository.BikeSpecDTO{
		ID:             model.ID,
		CanonicalSlug:  model.CanonicalSlug,
		Brand:          model.Brand,
		DisplayName:    model.DisplayName,
		Category:       model.Category,
		EngineCC:       int(model.EngineCc),
		IsVerified:     model.IsVerified,
		SourceProvider: model.SourceProvider,
		CreatedAt:      model.CreatedAt,
		Components:     make([]repository.SpecComponentDTO, len(comps)),
	}

	for i, c := range comps {
		dto.Components[i] = repository.SpecComponentDTO{
			ID:             c.ID,
			BikeModelID:    c.BikeModelID,
			Code:           c.Code,
			Name:           c.Name,
			Category:       c.Category,
			IntervalKM:     int(c.IntervalKm),
			IntervalMonths: int(c.IntervalMonths),
			UrgencyLevel:   c.UrgencyLevel,
			Notes:          c.Notes,
		}
	}

	return dto
}

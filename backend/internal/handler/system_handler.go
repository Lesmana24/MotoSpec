package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/motospec/backend/internal/llm"
	"github.com/motospec/backend/internal/repository"
)

type SystemHandler struct {
	store        repository.Store
	redisClient  redis.Cmdable
	telemetry    *repository.TelemetryStore
	orchestrator *llm.Orchestrator
}

func NewSystemHandler(
	store repository.Store,
	redisClient redis.Cmdable,
	telemetry *repository.TelemetryStore,
	orchestrator *llm.Orchestrator,
) *SystemHandler {
	return &SystemHandler{
		store:        store,
		redisClient:  redisClient,
		telemetry:    telemetry,
		orchestrator: orchestrator,
	}
}

// GetTelemetry handles GET /api/v1/system/telemetry
func (h *SystemHandler) GetTelemetry(c *gin.Context) {
	startTime := time.Now()
	snapshot := h.telemetry.Snapshot()

	state, consecutiveFailures := h.orchestrator.CircuitBreakerStats()

	stateStr := strings.ToLower(state.String())

	llmProviders := []gin.H{
		{
			"role":                 "primary",
			"name":                 h.orchestrator.PrimaryProviderName(),
			"circuit_state":        stateStr,
			"consecutive_failures": consecutiveFailures,
		},
		{
			"role":                 "fallback",
			"name":                 h.orchestrator.FallbackProviderName(),
			"circuit_state":        "closed",
			"consecutive_failures": 0,
		},
	}

	data := gin.H{
		"cache_hits":           snapshot.CacheHits,
		"cache_misses":         snapshot.CacheMisses,
		"cache_hit_ratio_pct":  snapshot.CacheHitRatioPct,
		"llm_primary_success":  snapshot.LLMPrimarySuccess,
		"llm_fallback_success": snapshot.LLMFallbackSuccess,
		"llm_total_failures":   snapshot.LLMTotalFailures,
		"llm_providers":        llmProviders,
	}

	meta := gin.H{
		"source":     "in_memory_metrics",
		"latency_ms": time.Since(startTime).Milliseconds(),
	}

	Success(c, http.StatusOK, data, meta)
}

// HealthCheck handles GET /healthz
func (h *SystemHandler) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	pgStatus := "connected"
	if err := h.store.Pool().Ping(ctx); err != nil {
		pgStatus = "disconnected: " + err.Error()
	}

	redisStatus := "connected"
	if err := h.redisClient.Ping(ctx).Err(); err != nil {
		redisStatus = "disconnected: " + err.Error()
	}

	status := "ok"
	httpCode := http.StatusOK
	if pgStatus != "connected" || redisStatus != "connected" {
		status = "degraded"
		httpCode = http.StatusServiceUnavailable
	}

	c.JSON(httpCode, gin.H{
		"status":   status,
		"postgres": pgStatus,
		"redis":    redisStatus,
	})
}

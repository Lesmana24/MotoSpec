package llm

import (
	"context"
	"errors"
)

type contextKey string

const (
	simulatedPrimaryOutageKey contextKey = "simulated_primary_outage"
)

// WithSimulatedPrimaryOutage injects a context flag forcing the orchestrator to bypass the primary LLM.
func WithSimulatedPrimaryOutage(ctx context.Context) context.Context {
	return context.WithValue(ctx, simulatedPrimaryOutageKey, true)
}

// IsSimulatedPrimaryOutage checks if the primary LLM outage simulation flag is set.
func IsSimulatedPrimaryOutage(ctx context.Context) bool {
	val, ok := ctx.Value(simulatedPrimaryOutageKey).(bool)
	return ok && val
}

// Sentinel errors for LLM operations.
var (
	ErrAllProvidersFailed = errors.New("llm: all providers failed")
	ErrSanityCheckFailed  = errors.New("llm: sanity check failed")
	ErrSimulatedOutage    = errors.New("llm: simulated primary outage")
	ErrCircuitBreakerOpen = errors.New("llm: circuit breaker is open")
)

// SpecGenerator defines the interface for an LLM provider generating motorcycle maintenance specifications.
type SpecGenerator interface {
	Name() string
	GenerateSpec(ctx context.Context, canonicalSlug string, rawQuery string) (*GeneratedBikeSpec, error)
}

// GeneratedBikeSpec represents the structured motorcycle profile output by an LLM provider.
type GeneratedBikeSpec struct {
	Brand       string                   `json:"brand" validate:"required,min=2,max=60"`
	DisplayName string                   `json:"display_name" validate:"required,min=3,max=120"`
	Category    string                   `json:"category" validate:"required,oneof=matic bebek sport naked trail cruiser"`
	EngineCC    int                      `json:"engine_cc" validate:"required,gte=50,lte=2500"`
	Components  []GeneratedSpecComponent `json:"components" validate:"required,min=4,max=20,dive"`
}

// GeneratedSpecComponent represents a single wear-and-tear component schedule.
type GeneratedSpecComponent struct {
	Code           string `json:"code" validate:"required,min=2,max=60"`
	Name           string `json:"name" validate:"required,min=2,max=100"`
	Category       string `json:"category" validate:"required,oneof=lubrication transmission brakes ignition filtration cooling tires"`
	IntervalKM     int    `json:"interval_km" validate:"required,gte=500,lte=100000"`
	IntervalMonths int    `json:"interval_months" validate:"gte=0,lte=60"`
	UrgencyLevel   string `json:"urgency_level" validate:"required,oneof=high medium low"`
	Notes          string `json:"notes" validate:"max=300"`
}

// GenerationResult encapsulates the generated spec and execution telemetry.
type GenerationResult struct {
	Spec              *GeneratedBikeSpec `json:"spec"`
	ProviderUsed      string             `json:"provider_used"`
	FallbackTriggered bool               `json:"fallback_triggered"`
	FallbackReason    string             `json:"fallback_reason,omitempty"`
}

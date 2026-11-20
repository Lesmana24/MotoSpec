package llm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sony/gobreaker"
)

// TelemetryRecorder allows the orchestrator to record success and failure counts.
type TelemetryRecorder interface {
	IncLLMPrimarySuccess() uint64
	IncLLMFallbackSuccess() uint64
	IncLLMTotalFailure() uint64
}

// Orchestrator coordinates primary and fallback LLM generation with circuit breaking and sanity validation.
type Orchestrator struct {
	primary   SpecGenerator
	fallback  SpecGenerator
	cb        *gobreaker.CircuitBreaker
	telemetry TelemetryRecorder
}

// NewOrchestrator creates a new Orchestrator with gobreaker configured on the primary provider.
func NewOrchestrator(primary SpecGenerator, fallback SpecGenerator, telemetry ...TelemetryRecorder) *Orchestrator {
	var rec TelemetryRecorder
	if len(telemetry) > 0 {
		rec = telemetry[0]
	}

	cbSettings := gobreaker.Settings{
		Name:        "PrimaryLLMProvider",
		MaxRequests: 1,
		Interval:    60 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	}

	return &Orchestrator{
		primary:   primary,
		fallback:  fallback,
		cb:        gobreaker.NewCircuitBreaker(cbSettings),
		telemetry: rec,
	}
}

// CircuitState returns the current state of the primary provider circuit breaker.
func (o *Orchestrator) CircuitState() gobreaker.State {
	return o.cb.State()
}

// GenerateSpec executes the fault-tolerant LLM resolution pipeline.
func (o *Orchestrator) GenerateSpec(ctx context.Context, canonicalSlug string, rawQuery string) (*GenerationResult, error) {
	// 1. Check for Outage Simulation Header / Context hook
	if IsSimulatedPrimaryOutage(ctx) {
		res, err := o.invokeFallback(ctx, canonicalSlug, rawQuery, "simulated_primary_outage")
		if err != nil {
			if o.telemetry != nil {
				o.telemetry.IncLLMTotalFailure()
			}
			return nil, fmt.Errorf("%w: fallback failed after simulated outage: %v", ErrAllProvidersFailed, err)
		}
		return res, nil
	}

	// 2. Attempt Primary Provider protected by gobreaker
	val, primaryErr := o.cb.Execute(func() (interface{}, error) {
		return o.primary.GenerateSpec(ctx, canonicalSlug, rawQuery)
	})

	if primaryErr == nil {
		if o.telemetry != nil {
			o.telemetry.IncLLMPrimarySuccess()
		}
		spec := val.(*GeneratedBikeSpec)
		return &GenerationResult{
			Spec:              spec,
			ProviderUsed:      o.primary.Name(),
			FallbackTriggered: false,
		}, nil
	}

	// 3. Primary Failed -> Determine fallback reason and invoke Fallback Provider
	var fallbackReason string
	if errors.Is(primaryErr, gobreaker.ErrOpenState) {
		fallbackReason = "circuit_breaker_open"
	} else if errors.Is(primaryErr, ErrSanityCheckFailed) {
		fallbackReason = fmt.Sprintf("primary_sanity_check_failed: %v", primaryErr)
	} else if errors.Is(primaryErr, context.DeadlineExceeded) {
		fallbackReason = "primary_timeout"
	} else {
		fallbackReason = fmt.Sprintf("primary_error: %v", primaryErr)
	}

	res, fallbackErr := o.invokeFallback(ctx, canonicalSlug, rawQuery, fallbackReason)
	if fallbackErr != nil {
		if o.telemetry != nil {
			o.telemetry.IncLLMTotalFailure()
		}
		return nil, fmt.Errorf("%w: primary failed (%v), fallback failed (%v)", ErrAllProvidersFailed, primaryErr, fallbackErr)
	}

	return res, nil
}

func (o *Orchestrator) invokeFallback(ctx context.Context, canonicalSlug string, rawQuery string, reason string) (*GenerationResult, error) {
	spec, err := o.fallback.GenerateSpec(ctx, canonicalSlug, rawQuery)
	if err != nil {
		return nil, err
	}

	if o.telemetry != nil {
		o.telemetry.IncLLMFallbackSuccess()
	}

	return &GenerationResult{
		Spec:              spec,
		ProviderUsed:      o.fallback.Name(),
		FallbackTriggered: true,
		FallbackReason:    reason,
	}, nil
}

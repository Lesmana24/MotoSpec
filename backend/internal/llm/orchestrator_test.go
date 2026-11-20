package llm_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/sony/gobreaker"

	"github.com/motospec/backend/internal/llm"
)

// mockGenerator is a controllable mock for SpecGenerator.
type mockGenerator struct {
	name        string
	specFn      func(ctx context.Context, canonicalSlug string, rawQuery string) (*llm.GeneratedBikeSpec, error)
	callCount   atomic.Int32
}

func (m *mockGenerator) Name() string {
	return m.name
}

func (m *mockGenerator) GenerateSpec(ctx context.Context, canonicalSlug string, rawQuery string) (*llm.GeneratedBikeSpec, error) {
	m.callCount.Add(1)
	if m.specFn != nil {
		return m.specFn(ctx, canonicalSlug, rawQuery)
	}
	return sampleValidSpec(), nil
}

func sampleValidSpec() *llm.GeneratedBikeSpec {
	return &llm.GeneratedBikeSpec{
		Brand:       "Honda",
		DisplayName: "Honda Vario 150",
		Category:    "matic",
		EngineCC:    150,
		Components: []llm.GeneratedSpecComponent{
			{
				Code:           "engine_oil",
				Name:           "Oli Mesin",
				Category:       "lubrication",
				IntervalKM:     4000,
				IntervalMonths: 4,
				UrgencyLevel:   "high",
				Notes:          "10W-30",
			},
			{
				Code:           "gear_oil",
				Name:           "Oli Gardan",
				Category:       "lubrication",
				IntervalKM:     8000,
				IntervalMonths: 8,
				UrgencyLevel:   "medium",
				Notes:          "Ganti tiap 2x ganti oli mesin",
			},
			{
				Code:           "v_belt",
				Name:           "V-Belt",
				Category:       "transmission",
				IntervalKM:     24000,
				IntervalMonths: 24,
				UrgencyLevel:   "high",
				Notes:          "Cek retakan",
			},
			{
				Code:           "spark_plug",
				Name:           "Busi",
				Category:       "ignition",
				IntervalKM:     8000,
				IntervalMonths: 8,
				UrgencyLevel:   "medium",
				Notes:          "CPR9EA-9",
			},
		},
	}
}

func TestOrchestrator_HappyPath_PrimarySuccess(t *testing.T) {
	primary := &mockGenerator{name: "gemini-2.5-flash"}
	fallback := &mockGenerator{name: "groq-llama-3.3-70b"}

	orch := llm.NewOrchestrator(primary, fallback)
	ctx := context.Background()

	result, err := orch.GenerateSpec(ctx, "honda-vario-150", "Vario 150")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if result.FallbackTriggered {
		t.Errorf("expected fallback_triggered to be false, got true")
	}
	if result.ProviderUsed != "gemini-2.5-flash" {
		t.Errorf("expected provider 'gemini-2.5-flash', got '%s'", result.ProviderUsed)
	}
	if primary.callCount.Load() != 1 {
		t.Errorf("expected primary called once, got %d", primary.callCount.Load())
	}
	if fallback.callCount.Load() != 0 {
		t.Errorf("expected fallback not called, got %d", fallback.callCount.Load())
	}
}

func TestOrchestrator_FallbackOnPrimaryError(t *testing.T) {
	primary := &mockGenerator{
		name: "gemini-2.5-flash",
		specFn: func(ctx context.Context, canonicalSlug string, rawQuery string) (*llm.GeneratedBikeSpec, error) {
			return nil, errors.New("upstream HTTP 500 internal server error")
		},
	}
	fallback := &mockGenerator{name: "groq-llama-3.3-70b"}

	orch := llm.NewOrchestrator(primary, fallback)
	ctx := context.Background()

	result, err := orch.GenerateSpec(ctx, "honda-vario-150", "Vario 150")
	if err != nil {
		t.Fatalf("expected fallback success, got error: %v", err)
	}

	if !result.FallbackTriggered {
		t.Errorf("expected fallback_triggered to be true")
	}
	if result.ProviderUsed != "groq-llama-3.3-70b" {
		t.Errorf("expected provider 'groq-llama-3.3-70b', got '%s'", result.ProviderUsed)
	}
	if primary.callCount.Load() != 1 {
		t.Errorf("expected primary called 1 time, got %d", primary.callCount.Load())
	}
	if fallback.callCount.Load() != 1 {
		t.Errorf("expected fallback called 1 time, got %d", fallback.callCount.Load())
	}
}

func TestOrchestrator_FallbackOnPrimaryTimeout(t *testing.T) {
	primary := &mockGenerator{
		name: "gemini-2.5-flash",
		specFn: func(ctx context.Context, canonicalSlug string, rawQuery string) (*llm.GeneratedBikeSpec, error) {
			return nil, context.DeadlineExceeded
		},
	}
	fallback := &mockGenerator{name: "groq-llama-3.3-70b"}

	orch := llm.NewOrchestrator(primary, fallback)
	ctx := context.Background()

	result, err := orch.GenerateSpec(ctx, "honda-vario-150", "Vario 150")
	if err != nil {
		t.Fatalf("expected fallback success, got error: %v", err)
	}

	if !result.FallbackTriggered {
		t.Errorf("expected fallback_triggered to be true")
	}
	if result.FallbackReason != "primary_timeout" {
		t.Errorf("expected reason 'primary_timeout', got '%s'", result.FallbackReason)
	}
}

func TestOrchestrator_FallbackOnSanityCheckFailure(t *testing.T) {
	primary := &mockGenerator{
		name: "gemini-2.5-flash",
		specFn: func(ctx context.Context, canonicalSlug string, rawQuery string) (*llm.GeneratedBikeSpec, error) {
			return nil, errors.New("llm: sanity check failed: missing mandatory 'engine_oil' component")
		},
	}
	fallback := &mockGenerator{name: "groq-llama-3.3-70b"}

	orch := llm.NewOrchestrator(primary, fallback)
	ctx := context.Background()

	result, err := orch.GenerateSpec(ctx, "honda-vario-150", "Vario 150")
	if err != nil {
		t.Fatalf("expected fallback success, got error: %v", err)
	}

	if !result.FallbackTriggered {
		t.Errorf("expected fallback_triggered to be true")
	}
}

func TestOrchestrator_FallbackOnSimulatedOutage(t *testing.T) {
	primary := &mockGenerator{name: "gemini-2.5-flash"}
	fallback := &mockGenerator{name: "groq-llama-3.3-70b"}

	orch := llm.NewOrchestrator(primary, fallback)
	ctx := llm.WithSimulatedPrimaryOutage(context.Background())

	result, err := orch.GenerateSpec(ctx, "honda-vario-150", "Vario 150")
	if err != nil {
		t.Fatalf("expected fallback success, got error: %v", err)
	}

	if !result.FallbackTriggered {
		t.Errorf("expected fallback_triggered to be true")
	}
	if result.FallbackReason != "simulated_primary_outage" {
		t.Errorf("expected reason 'simulated_primary_outage', got '%s'", result.FallbackReason)
	}
	if primary.callCount.Load() != 0 {
		t.Errorf("primary should have been bypassed (callCount 0), got %d", primary.callCount.Load())
	}
	if fallback.callCount.Load() != 1 {
		t.Errorf("fallback should have been called once, got %d", fallback.callCount.Load())
	}
}

func TestOrchestrator_CircuitBreaker_TripsAfter3Failures(t *testing.T) {
	primary := &mockGenerator{
		name: "gemini-2.5-flash",
		specFn: func(ctx context.Context, canonicalSlug string, rawQuery string) (*llm.GeneratedBikeSpec, error) {
			return nil, errors.New("gemini network timeout")
		},
	}
	fallback := &mockGenerator{name: "groq-llama-3.3-70b"}

	orch := llm.NewOrchestrator(primary, fallback)
	ctx := context.Background()

	// 3 consecutive failures to trip the circuit breaker
	for i := 0; i < 3; i++ {
		res, err := orch.GenerateSpec(ctx, "honda-vario-150", "Vario 150")
		if err != nil {
			t.Fatalf("call %d: expected fallback success, got %v", i+1, err)
		}
		if !res.FallbackTriggered {
			t.Errorf("call %d: expected fallback triggered", i+1)
		}
	}

	if orch.CircuitState() != gobreaker.StateOpen {
		t.Fatalf("expected circuit breaker state to be Open after 3 failures, got %v", orch.CircuitState())
	}

	primaryCallsBefore4th := primary.callCount.Load()
	if primaryCallsBefore4th != 3 {
		t.Fatalf("expected primary to have been called 3 times, got %d", primaryCallsBefore4th)
	}

	// 4th call: circuit breaker is Open, so primary.specFn should NOT be called at all
	res4, err := orch.GenerateSpec(ctx, "honda-vario-150", "Vario 150")
	if err != nil {
		t.Fatalf("call 4: expected fallback success, got %v", err)
	}
	if !res4.FallbackTriggered {
		t.Errorf("call 4: expected fallback triggered")
	}
	if res4.FallbackReason != "circuit_breaker_open" {
		t.Errorf("call 4: expected reason 'circuit_breaker_open', got '%s'", res4.FallbackReason)
	}
	if primary.callCount.Load() != 3 {
		t.Errorf("primary should NOT have been called while circuit breaker was Open (expected 3, got %d)", primary.callCount.Load())
	}
}

func TestOrchestrator_AllProvidersFailed(t *testing.T) {
	primary := &mockGenerator{
		name: "gemini-2.5-flash",
		specFn: func(ctx context.Context, canonicalSlug string, rawQuery string) (*llm.GeneratedBikeSpec, error) {
			return nil, errors.New("primary down")
		},
	}
	fallback := &mockGenerator{
		name: "groq-llama-3.3-70b",
		specFn: func(ctx context.Context, canonicalSlug string, rawQuery string) (*llm.GeneratedBikeSpec, error) {
			return nil, errors.New("fallback down")
		},
	}

	orch := llm.NewOrchestrator(primary, fallback)
	ctx := context.Background()

	_, err := orch.GenerateSpec(ctx, "honda-vario-150", "Vario 150")
	if err == nil {
		t.Fatalf("expected error when all providers fail, got nil")
	}
	if !errors.Is(err, llm.ErrAllProvidersFailed) {
		t.Errorf("expected ErrAllProvidersFailed, got %v", err)
	}
}

// Tests for Validator & Markdown Stripper
func TestValidator_ValidJSON(t *testing.T) {
	raw := `{
		"brand": "Yamaha",
		"display_name": "Yamaha NMAX 155",
		"category": "matic",
		"engine_cc": 155,
		"components": [
			{"code": "engine_oil", "name": "Oli Mesin", "category": "lubrication", "interval_km": 4000, "interval_months": 4, "urgency_level": "high", "notes": ""},
			{"code": "gear_oil", "name": "Oli Gardan", "category": "lubrication", "interval_km": 8000, "interval_months": 8, "urgency_level": "medium", "notes": ""},
			{"code": "v_belt", "name": "V-Belt", "category": "transmission", "interval_km": 25000, "interval_months": 24, "urgency_level": "high", "notes": ""},
			{"code": "coolant", "name": "Air Radiator", "category": "cooling", "interval_km": 12000, "interval_months": 12, "urgency_level": "medium", "notes": ""}
		]
	}`

	spec, err := llm.ValidateAndSanitizeSpec([]byte(raw))
	if err != nil {
		t.Fatalf("expected valid spec, got error: %v", err)
	}
	if spec.Brand != "Yamaha" || spec.EngineCC != 155 {
		t.Errorf("unexpected spec content: %+v", spec)
	}
}

func TestValidator_MarkdownCodeFences(t *testing.T) {
	fenced := "```json\n" + `{
		"brand": "Yamaha",
		"display_name": "Yamaha NMAX 155",
		"category": "matic",
		"engine_cc": 155,
		"components": [
			{"code": "engine_oil", "name": "Oli Mesin", "category": "lubrication", "interval_km": 4000, "interval_months": 4, "urgency_level": "high", "notes": ""},
			{"code": "gear_oil", "name": "Oli Gardan", "category": "lubrication", "interval_km": 8000, "interval_months": 8, "urgency_level": "medium", "notes": ""},
			{"code": "v_belt", "name": "V-Belt", "category": "transmission", "interval_km": 25000, "interval_months": 24, "urgency_level": "high", "notes": ""},
			{"code": "coolant", "name": "Air Radiator", "category": "cooling", "interval_km": 12000, "interval_months": 12, "urgency_level": "medium", "notes": ""}
		]
	}` + "\n```"

	spec, err := llm.ValidateAndSanitizeSpec([]byte(fenced))
	if err != nil {
		t.Fatalf("expected success with markdown fences stripped, got: %v", err)
	}
	if spec.Brand != "Yamaha" {
		t.Errorf("expected brand 'Yamaha', got '%s'", spec.Brand)
	}
}

func TestValidator_MissingEngineOil(t *testing.T) {
	raw := `{
		"brand": "Yamaha",
		"display_name": "Yamaha NMAX 155",
		"category": "matic",
		"engine_cc": 155,
		"components": [
			{"code": "brake_pads", "name": "Kampas Rem", "category": "brakes", "interval_km": 10000, "interval_months": 12, "urgency_level": "high", "notes": ""},
			{"code": "gear_oil", "name": "Oli Gardan", "category": "lubrication", "interval_km": 8000, "interval_months": 8, "urgency_level": "medium", "notes": ""},
			{"code": "v_belt", "name": "V-Belt", "category": "transmission", "interval_km": 25000, "interval_months": 24, "urgency_level": "high", "notes": ""},
			{"code": "coolant", "name": "Air Radiator", "category": "cooling", "interval_km": 12000, "interval_months": 12, "urgency_level": "medium", "notes": ""}
		]
	}`

	_, err := llm.ValidateAndSanitizeSpec([]byte(raw))
	if err == nil {
		t.Fatalf("expected error when engine_oil is missing, got nil")
	}
	if !errors.Is(err, llm.ErrSanityCheckFailed) {
		t.Errorf("expected ErrSanityCheckFailed, got %v", err)
	}
}

func TestValidator_DuplicateComponentCodes(t *testing.T) {
	raw := `{
		"brand": "Yamaha",
		"display_name": "Yamaha NMAX 155",
		"category": "matic",
		"engine_cc": 155,
		"components": [
			{"code": "engine_oil", "name": "Oli Mesin", "category": "lubrication", "interval_km": 4000, "interval_months": 4, "urgency_level": "high", "notes": ""},
			{"code": "engine_oil", "name": "Oli Mesin 2", "category": "lubrication", "interval_km": 4000, "interval_months": 4, "urgency_level": "high", "notes": ""},
			{"code": "v_belt", "name": "V-Belt", "category": "transmission", "interval_km": 25000, "interval_months": 24, "urgency_level": "high", "notes": ""},
			{"code": "coolant", "name": "Air Radiator", "category": "cooling", "interval_km": 12000, "interval_months": 12, "urgency_level": "medium", "notes": ""}
		]
	}`

	_, err := llm.ValidateAndSanitizeSpec([]byte(raw))
	if err == nil {
		t.Fatalf("expected error on duplicate component codes, got nil")
	}
	if !errors.Is(err, llm.ErrSanityCheckFailed) {
		t.Errorf("expected ErrSanityCheckFailed, got %v", err)
	}
}

func TestValidator_IntervalKMLow(t *testing.T) {
	raw := `{
		"brand": "Yamaha",
		"display_name": "Yamaha NMAX 155",
		"category": "matic",
		"engine_cc": 155,
		"components": [
			{"code": "engine_oil", "name": "Oli Mesin", "category": "lubrication", "interval_km": 100, "interval_months": 4, "urgency_level": "high", "notes": ""},
			{"code": "gear_oil", "name": "Oli Gardan", "category": "lubrication", "interval_km": 8000, "interval_months": 8, "urgency_level": "medium", "notes": ""},
			{"code": "v_belt", "name": "V-Belt", "category": "transmission", "interval_km": 25000, "interval_months": 24, "urgency_level": "high", "notes": ""},
			{"code": "coolant", "name": "Air Radiator", "category": "cooling", "interval_km": 12000, "interval_months": 12, "urgency_level": "medium", "notes": ""}
		]
	}`

	_, err := llm.ValidateAndSanitizeSpec([]byte(raw))
	if err == nil {
		t.Fatalf("expected validation error when interval_km < 500, got nil")
	}
}

// HTTP Mock tests for PrimaryProviderAdapter and FallbackProviderAdapter
func TestPrimaryProvider_HTTP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := `{
			"candidates": [
				{
					"content": {
						"parts": [
							{
								"text": "{\"brand\":\"Honda\",\"display_name\":\"Honda Beat\",\"category\":\"matic\",\"engine_cc\":110,\"components\":[{\"code\":\"engine_oil\",\"name\":\"Oli Mesin\",\"category\":\"lubrication\",\"interval_km\":4000,\"interval_months\":4,\"urgency_level\":\"high\",\"notes\":\"\"},{\"code\":\"gear_oil\",\"name\":\"Oli Gardan\",\"category\":\"lubrication\",\"interval_km\":8000,\"interval_months\":8,\"urgency_level\":\"medium\",\"notes\":\"\"},{\"code\":\"spark_plug\",\"name\":\"Busi\",\"category\":\"ignition\",\"interval_km\":8000,\"interval_months\":8,\"urgency_level\":\"medium\",\"notes\":\"\"},{\"code\":\"air_filter\",\"name\":\"Filter Udara\",\"category\":\"filtration\",\"interval_km\":16000,\"interval_months\":16,\"urgency_level\":\"low\",\"notes\":\"\"}]}"
							}
						]
					}
				}
			]
		}`
		w.Write([]byte(response))
	}))
	defer ts.Close()

	provider := llm.NewPrimaryProvider("dummy-key", ts.URL, ts.Client())
	spec, err := provider.GenerateSpec(context.Background(), "honda-beat", "Honda Beat")
	if err != nil {
		t.Fatalf("expected successful primary provider call, got: %v", err)
	}

	if spec.Brand != "Honda" || spec.DisplayName != "Honda Beat" {
		t.Errorf("unexpected spec result: %+v", spec)
	}
}

func TestFallbackProvider_HTTP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Authorization header with Bearer token, got '%s'", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		response := `{
			"choices": [
				{
					"message": {
						"content": "{\"brand\":\"Suzuki\",\"display_name\":\"Suzuki Satria F150\",\"category\":\"bebek\",\"engine_cc\":150,\"components\":[{\"code\":\"engine_oil\",\"name\":\"Oli Mesin\",\"category\":\"lubrication\",\"interval_km\":3000,\"interval_months\":3,\"urgency_level\":\"high\",\"notes\":\"\"},{\"code\":\"spark_plug\",\"name\":\"Busi\",\"category\":\"ignition\",\"interval_km\":6000,\"interval_months\":6,\"urgency_level\":\"medium\",\"notes\":\"\"},{\"code\":\"oil_filter\",\"name\":\"Filter Oli\",\"category\":\"filtration\",\"interval_km\":6000,\"interval_months\":6,\"urgency_level\":\"medium\",\"notes\":\"\"},{\"code\":\"brake_pads_front\",\"name\":\"Kampas Rem Depan\",\"category\":\"brakes\",\"interval_km\":10000,\"interval_months\":12,\"urgency_level\":\"high\",\"notes\":\"\"}]}"
					}
				}
			]
		}`
		w.Write([]byte(response))
	}))
	defer ts.Close()

	provider := llm.NewFallbackProvider("test-api-key", ts.URL, "llama-3.3-70b-versatile", ts.Client())
	spec, err := provider.GenerateSpec(context.Background(), "suzuki-satria-f150", "Satria F150")
	if err != nil {
		t.Fatalf("expected successful fallback provider call, got: %v", err)
	}

	if spec.Brand != "Suzuki" || spec.DisplayName != "Suzuki Satria F150" {
		t.Errorf("unexpected spec result: %+v", spec)
	}
}

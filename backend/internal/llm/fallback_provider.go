package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	DefaultGroqBaseURL = "https://api.groq.com/openai/v1/chat/completions"
)

// FallbackProviderAdapter implements SpecGenerator for Groq / OpenAI-compatible chat completion APIs.
type FallbackProviderAdapter struct {
	name       string
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	timeout    time.Duration
}

// NewFallbackProvider creates a new FallbackProviderAdapter instance.
func NewFallbackProvider(apiKey string, baseURL string, model string, httpClient *http.Client) *FallbackProviderAdapter {
	if baseURL == "" {
		baseURL = DefaultGroqBaseURL
	}
	if model == "" {
		model = "llama-3.3-70b-versatile"
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &FallbackProviderAdapter{
		name:       "groq-llama-3.3-70b",
		apiKey:     apiKey,
		baseURL:    baseURL,
		model:      model,
		httpClient: httpClient,
		timeout:    DefaultProviderTimeout,
	}
}

func (f *FallbackProviderAdapter) Name() string {
	return f.name
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatRequest struct {
	Model          string              `json:"model"`
	Messages       []openAIChatMessage `json:"messages"`
	ResponseFormat struct {
		Type string `json:"type"`
	} `json:"response_format"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (f *FallbackProviderAdapter) GenerateSpec(ctx context.Context, canonicalSlug string, rawQuery string) (*GeneratedBikeSpec, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	systemPrompt := `You are an expert motorcycle specification service.
Return a STRICT JSON object representing official service schedule.
JSON Schema:
{
  "brand": "string (e.g. Honda, Yamaha)",
  "display_name": "string (e.g. Honda Vario 150)",
  "category": "one of: matic, bebek, sport, naked, trail, cruiser",
  "engine_cc": integer (50-2500),
  "components": [
    {
      "code": "unique snake_case (e.g. engine_oil, gear_oil, v_belt, brake_pads_front)",
      "name": "display name in Indonesian or English (e.g. Oli Mesin)",
      "category": "one of: lubrication, transmission, brakes, ignition, filtration, cooling, tires",
      "interval_km": integer (>= 500),
      "interval_months": integer (>= 0),
      "urgency_level": "one of: high, medium, low",
      "notes": "string specification recommendation"
    }
  ]
}
Mandatory rules:
1. Must include at least 4 components.
2. Must include 'engine_oil'.
3. All codes must be unique.`

	userPrompt := fmt.Sprintf(`Generate full maintenance specifications for motorcycle "%s" with canonical slug "%s".`, rawQuery, canonicalSlug)

	reqPayload := openAIChatRequest{
		Model: f.model,
		Messages: []openAIChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
	reqPayload.ResponseFormat.Type = "json_object"

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal openai/groq request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctxWithTimeout, http.MethodPost, f.baseURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if f.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+f.apiKey)
	}

	resp, err := f.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("fallback http error: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read fallback response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fallback api returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var chatResp openAIChatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to parse fallback response wrapper: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("fallback returned no choices")
	}

	rawText := chatResp.Choices[0].Message.Content
	return ValidateAndSanitizeSpec([]byte(rawText))
}

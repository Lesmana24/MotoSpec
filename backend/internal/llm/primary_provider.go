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
	DefaultGeminiBaseURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent"
	DefaultProviderTimeout = 4 * time.Second
)

// PrimaryProviderAdapter implements SpecGenerator for the Gemini REST API.
type PrimaryProviderAdapter struct {
	name       string
	apiKey     string
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

// NewPrimaryProvider creates a new PrimaryProviderAdapter instance.
func NewPrimaryProvider(apiKey string, baseURL string, httpClient *http.Client) *PrimaryProviderAdapter {
	if baseURL == "" {
		baseURL = DefaultGeminiBaseURL
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &PrimaryProviderAdapter{
		name:       "gemini-2.5-flash",
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: httpClient,
		timeout:    DefaultProviderTimeout,
	}
}

func (p *PrimaryProviderAdapter) Name() string {
	return p.name
}

// Gemini API request schemas
type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	ResponseMimeType string `json:"responseMimeType"`
}

type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	GenerationConfig geminiGenerationConfig `json:"generationConfig"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (p *PrimaryProviderAdapter) GenerateSpec(ctx context.Context, canonicalSlug string, rawQuery string) (*GeneratedBikeSpec, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	prompt := fmt.Sprintf(`You are a motorcycle maintenance specification expert.
Return a STRICT JSON object representing official service schedule for: "%s" (slug: "%s").
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
3. All codes must be unique.`, rawQuery, canonicalSlug)

	reqPayload := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: geminiGenerationConfig{
			ResponseMimeType: "application/json",
		},
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gemini request: %w", err)
	}

	url := p.baseURL
	if p.apiKey != "" {
		url = fmt.Sprintf("%s?key=%s", url, p.apiKey)
	}

	httpReq, err := http.NewRequestWithContext(ctxWithTimeout, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini http error: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read gemini response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini api returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(respBytes, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse gemini response wrapper: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini returned no content candidates")
	}

	rawText := geminiResp.Candidates[0].Content.Parts[0].Text
	return ValidateAndSanitizeSpec([]byte(rawText))
}

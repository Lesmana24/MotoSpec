package llm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	validate = validator.New()
	codeFenceRegex = regexp.MustCompile("(?s)^```(?:json)?\\s*([\\s\\S]*?)\\s*```$")
)

// StripMarkdownFences removes markdown code fences (```json ... ``` or ``` ... ```) from raw LLM output.
func StripMarkdownFences(raw string) string {
	trimmed := strings.TrimSpace(raw)
	matches := codeFenceRegex.FindStringSubmatch(trimmed)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return trimmed
}

// ValidateAndSanitizeSpec parses, validates, and runs domain sanity checks on raw LLM JSON.
func ValidateAndSanitizeSpec(raw []byte) (*GeneratedBikeSpec, error) {
	cleanStr := StripMarkdownFences(string(raw))
	if cleanStr == "" {
		return nil, fmt.Errorf("%w: empty payload after stripping markdown fences", ErrSanityCheckFailed)
	}

	var spec GeneratedBikeSpec
	if err := json.Unmarshal([]byte(cleanStr), &spec); err != nil {
		return nil, fmt.Errorf("%w: json unmarshal error: %v", ErrSanityCheckFailed, err)
	}

	// 1. Struct validation tags
	if err := validate.Struct(&spec); err != nil {
		return nil, fmt.Errorf("%w: struct validation failed: %v", ErrSanityCheckFailed, err)
	}

	// 2. Domain Sanity Check: Must contain 'engine_oil'
	hasEngineOil := false
	seenCodes := make(map[string]struct{}, len(spec.Components))

	for _, comp := range spec.Components {
		code := strings.TrimSpace(strings.ToLower(comp.Code))
		if code == "engine_oil" {
			hasEngineOil = true
		}

		// 3. Domain Sanity Check: Unique component codes
		if _, exists := seenCodes[code]; exists {
			return nil, fmt.Errorf("%w: duplicate component code '%s'", ErrSanityCheckFailed, comp.Code)
		}
		seenCodes[code] = struct{}{}
	}

	if !hasEngineOil {
		return nil, fmt.Errorf("%w: missing mandatory 'engine_oil' component", ErrSanityCheckFailed)
	}

	return &spec, nil
}

package service_test

import (
	"testing"

	"github.com/motospec/backend/internal/service"
)

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Trims whitespace and handles cc",
			input:    "  Vario 150cc  ",
			expected: "honda-vario-150",
		},
		{
			name:     "Already has brand and cc",
			input:    "Yamaha NMAX 155cc",
			expected: "yamaha-nmax-155",
		},
		{
			name:     "Punctuation removal",
			input:    "Beat F.I. 110 cc!",
			expected: "honda-beat-fi-110",
		},
		{
			name:     "Brand alias prefixing without brand in input",
			input:    "aerox 155",
			expected: "yamaha-aerox-155",
		},
		{
			name:     "Suzuki alias without brand",
			input:    "satria fu 150",
			expected: "suzuki-satria-fu-150",
		},
		{
			name:     "Kawasaki alias without brand",
			input:    "ninja zx25r",
			expected: "kawasaki-ninja-zx25r",
		},
		{
			name:     "Vespa alias without brand",
			input:    "sprint 150",
			expected: "vespa-sprint-150",
		},
		{
			name:     "Empty input",
			input:    "   ",
			expected: "",
		},
		{
			name:     "Multiple spaces and hyphens",
			input:    "Honda --- CBR   250  RR",
			expected: "honda-cbr-250-rr",
		},
		{
			name:     "No alias match keeps sanitized slug",
			input:    "Harley Fat Bob 114",
			expected: "harley-fat-bob-114",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := service.NormalizeQuery(tt.input)
			if actual != tt.expected {
				t.Errorf("NormalizeQuery(%q) = %q, expected %q", tt.input, actual, tt.expected)
			}
		})
	}
}

package repository

import (
	"time"

	"github.com/google/uuid"
)

// BikeSpecDTO represents the cached specification of a motorcycle model.
type BikeSpecDTO struct {
	ID             uuid.UUID          `json:"id"`
	CanonicalSlug  string             `json:"canonical_slug"`
	Brand          string             `json:"brand"`
	DisplayName    string             `json:"display_name"`
	Category       string             `json:"category"`
	EngineCC       int                `json:"engine_cc"`
	IsVerified     bool               `json:"is_verified"`
	SourceProvider string             `json:"source_provider"`
	CreatedAt      time.Time          `json:"created_at"`
	Components     []SpecComponentDTO `json:"components"`
}

// SpecComponentDTO represents a maintenance component specification.
type SpecComponentDTO struct {
	ID             uuid.UUID `json:"id"`
	BikeModelID    uuid.UUID `json:"bike_model_id"`
	Code           string    `json:"code"`
	Name           string    `json:"name"`
	Category       string    `json:"category"`
	IntervalKM     int       `json:"interval_km"`
	IntervalMonths int       `json:"interval_months"`
	UrgencyLevel   string    `json:"urgency_level"`
	Notes          string    `json:"notes"`
}

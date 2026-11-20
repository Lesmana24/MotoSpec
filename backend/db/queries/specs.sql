-- name: GetBikeModelBySlug :one
SELECT id, canonical_slug, brand, display_name, category, engine_cc, is_verified, source_provider, created_at
FROM bike_models
WHERE canonical_slug = $1 LIMIT 1;

-- name: GetBikeModelByID :one
SELECT id, canonical_slug, brand, display_name, category, engine_cc, is_verified, source_provider, created_at
FROM bike_models
WHERE id = $1 LIMIT 1;

-- name: ListBikeModels :many
SELECT id, canonical_slug, brand, display_name, category, engine_cc, is_verified, source_provider, created_at
FROM bike_models
ORDER BY display_name ASC;

-- name: CreateBikeModel :one
INSERT INTO bike_models (
    canonical_slug, brand, display_name, category, engine_cc, is_verified, source_provider
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING id, canonical_slug, brand, display_name, category, engine_cc, is_verified, source_provider, created_at;

-- name: ListSpecComponentsByModelID :many
SELECT id, bike_model_id, code, name, category, interval_km, interval_months, urgency_level, notes
FROM bike_spec_components
WHERE bike_model_id = $1
ORDER BY interval_km ASC, code ASC;

-- name: CreateSpecComponent :one
INSERT INTO bike_spec_components (
    bike_model_id, code, name, category, interval_km, interval_months, urgency_level, notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING id, bike_model_id, code, name, category, interval_km, interval_months, urgency_level, notes;

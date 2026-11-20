-- name: ListUserBikes :many
SELECT 
    ub.id,
    ub.user_id,
    ub.bike_model_id,
    ub.nickname,
    ub.current_odometer_km,
    ub.daily_estimate_km,
    ub.auto_increment_enabled,
    ub.last_odometer_sync_at,
    ub.created_at,
    ub.updated_at,
    bm.display_name AS model_display_name,
    bm.category AS model_category
FROM user_bikes ub
JOIN bike_models bm ON ub.bike_model_id = bm.id
WHERE ub.user_id = $1
ORDER BY ub.created_at DESC;

-- name: GetUserBikeByID :one
SELECT 
    ub.id,
    ub.user_id,
    ub.bike_model_id,
    ub.nickname,
    ub.current_odometer_km,
    ub.daily_estimate_km,
    ub.auto_increment_enabled,
    ub.last_odometer_sync_at,
    ub.created_at,
    ub.updated_at,
    bm.display_name AS model_display_name,
    bm.category AS model_category,
    bm.engine_cc AS model_engine_cc
FROM user_bikes ub
JOIN bike_models bm ON ub.bike_model_id = bm.id
WHERE ub.id = $1 AND ub.user_id = $2
LIMIT 1;

-- name: CreateUserBike :one
INSERT INTO user_bikes (
    user_id, bike_model_id, nickname, current_odometer_km, daily_estimate_km, auto_increment_enabled
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: UpdateUserBikeOdometer :one
UPDATE user_bikes
SET current_odometer_km = $3,
    daily_estimate_km = $4,
    auto_increment_enabled = $5,
    last_odometer_sync_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: BumpUserBikeOdometerIfHigher :one
UPDATE user_bikes
SET current_odometer_km = GREATEST(current_odometer_km, $2),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: BatchAutoIncrementOdometers :execrows
UPDATE user_bikes
SET current_odometer_km = current_odometer_km + daily_estimate_km,
    last_odometer_sync_at = NOW(),
    updated_at = NOW()
WHERE auto_increment_enabled = TRUE
  AND daily_estimate_km > 0
  AND last_odometer_sync_at < CURRENT_DATE;

-- name: GetUserComponentStatesWithSpecs :many
SELECT 
    ucs.id AS component_state_id,
    ucs.user_bike_id,
    ucs.spec_component_id,
    ucs.last_replaced_km,
    ucs.last_replaced_at,
    ucs.updated_at,
    bsc.code,
    bsc.name,
    bsc.category,
    bsc.interval_km,
    bsc.interval_months,
    bsc.urgency_level,
    bsc.notes
FROM user_component_states ucs
JOIN bike_spec_components bsc ON ucs.spec_component_id = bsc.id
WHERE ucs.user_bike_id = $1
ORDER BY bsc.interval_km ASC, bsc.code ASC;

-- name: UpsertUserComponentState :one
INSERT INTO user_component_states (
    user_bike_id, spec_component_id, last_replaced_km, last_replaced_at
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (user_bike_id, spec_component_id)
DO UPDATE SET
    last_replaced_km = EXCLUDED.last_replaced_km,
    last_replaced_at = EXCLUDED.last_replaced_at,
    updated_at = NOW()
RETURNING *;

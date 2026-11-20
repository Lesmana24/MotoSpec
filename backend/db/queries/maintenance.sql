-- name: CreateMaintenanceLog :one
INSERT INTO maintenance_logs (
    user_bike_id, spec_component_id, odometer_km, serviced_at, cost_idr, notes
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: ListMaintenanceLogsByBike :many
SELECT 
    ml.id,
    ml.user_bike_id,
    ml.spec_component_id,
    ml.odometer_km,
    ml.serviced_at,
    ml.cost_idr,
    ml.notes,
    ml.created_at,
    bsc.code AS component_code,
    bsc.name AS component_name
FROM maintenance_logs ml
JOIN bike_spec_components bsc ON ml.spec_component_id = bsc.id
WHERE ml.user_bike_id = $1
ORDER BY ml.serviced_at DESC, ml.created_at DESC;

-- name: ResetComponentStateOnMaintenance :one
UPDATE user_component_states
SET last_replaced_km = $3,
    last_replaced_at = $4,
    updated_at = NOW()
WHERE user_bike_id = $1 AND spec_component_id = $2
RETURNING *;

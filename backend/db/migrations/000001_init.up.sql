CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. Master Motorcycle Catalog (Seeded by LLM or Admin)
CREATE TABLE bike_models (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    canonical_slug VARCHAR(120) UNIQUE NOT NULL,
    brand VARCHAR(60) NOT NULL,
    display_name VARCHAR(120) NOT NULL,
    category VARCHAR(40) NOT NULL,
    engine_cc INT NOT NULL CHECK (engine_cc > 0),
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    source_provider VARCHAR(80) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bike_models_slug ON bike_models(canonical_slug);

-- 2. Master Component Intervals per Bike Model
CREATE TABLE bike_spec_components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bike_model_id UUID NOT NULL REFERENCES bike_models(id) ON DELETE CASCADE,
    code VARCHAR(60) NOT NULL,
    name VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL,
    interval_km INT NOT NULL CHECK (interval_km >= 500),
    interval_months INT NOT NULL DEFAULT 0 CHECK (interval_months >= 0),
    urgency_level VARCHAR(20) NOT NULL DEFAULT 'medium',
    notes TEXT NOT NULL DEFAULT '',
    UNIQUE (bike_model_id, code)
);

CREATE INDEX idx_spec_components_model_id ON bike_spec_components(bike_model_id);

-- 3. User Garage Motorcycles
CREATE TABLE user_bikes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(100) NOT NULL,
    bike_model_id UUID NOT NULL REFERENCES bike_models(id) ON DELETE RESTRICT,
    nickname VARCHAR(100) NOT NULL,
    current_odometer_km INT NOT NULL CHECK (current_odometer_km >= 0),
    daily_estimate_km INT NOT NULL DEFAULT 0 CHECK (daily_estimate_km >= 0),
    auto_increment_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    last_odometer_sync_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_bikes_user_id ON user_bikes(user_id);

-- 4. Current Component Baseline per User Bike
CREATE TABLE user_component_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_bike_id UUID NOT NULL REFERENCES user_bikes(id) ON DELETE CASCADE,
    spec_component_id UUID NOT NULL REFERENCES bike_spec_components(id) ON DELETE CASCADE,
    last_replaced_km INT NULL CHECK (last_replaced_km >= 0),
    last_replaced_at TIMESTAMPTZ NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_bike_id, spec_component_id)
);

CREATE INDEX idx_user_comp_states_bike_id ON user_component_states(user_bike_id);

-- 5. Historical Maintenance Audit Logs
CREATE TABLE maintenance_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_bike_id UUID NOT NULL REFERENCES user_bikes(id) ON DELETE CASCADE,
    spec_component_id UUID NOT NULL REFERENCES bike_spec_components(id) ON DELETE CASCADE,
    odometer_km INT NOT NULL CHECK (odometer_km >= 0),
    serviced_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cost_idr INT NULL CHECK (cost_idr >= 0),
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_maintenance_logs_bike_id ON maintenance_logs(user_bike_id);
CREATE INDEX idx_maintenance_logs_serviced_at ON maintenance_logs(serviced_at DESC);

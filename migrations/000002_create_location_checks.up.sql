CREATE TABLE IF NOT EXISTS location_checks (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    location GEOGRAPHY(Point) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_location_checks_created_at ON location_checks(created_at);
CREATE INDEX IF NOT EXISTS idx_location_checks_user_id_created_at ON location_checks(user_id, created_at);

CREATE TABLE IF NOT EXISTS location_check_incidents (
    check_id BIGINT NOT NULL REFERENCES location_checks(id) ON DELETE CASCADE,
    incident_id BIGINT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    PRIMARY KEY (check_id, incident_id)
);

CREATE INDEX IF NOT EXISTS idx_location_check_incidents_incident_id ON location_check_incidents(incident_id);
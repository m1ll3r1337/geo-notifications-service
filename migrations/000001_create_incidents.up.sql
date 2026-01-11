CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE incidents (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    center GEOGRAPHY(Point) NOT NULL,
    radius INTEGER NOT NULL CHECK (radius > 0),
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_incidents_active ON incidents(active) WHERE active = true;
CREATE INDEX idx_incidents_center ON incidents USING GIST(center) WHERE active = true;
CREATE INDEX idx_incidents_created_at ON incidents(created_at);

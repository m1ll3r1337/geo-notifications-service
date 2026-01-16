CREATE TABLE IF NOT EXISTS webhook_outbox (
    id              BIGSERIAL PRIMARY KEY,
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL,
    status          TEXT NOT NULL CHECK (status IN ('pending','dispatched','dead')),
    attempts        INT NOT NULL DEFAULT 0,
    processing_until TIMESTAMPTZ NULL,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error      TEXT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhook_outbox_pending ON webhook_outbox(status, next_attempt_at);
CREATE INDEX IF NOT EXISTS idx_outbox_processing_until ON webhook_outbox(status, processing_until);
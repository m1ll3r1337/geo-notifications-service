package outboxdb

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"

	dberrs "github.com/m1ll3r1337/geo-notifications-service/internal/platform/db/errs"
)

type Repository struct {
	exec sqlx.ExtContext
}

func New(exec sqlx.ExtContext) *Repository { return &Repository{exec: exec} }

type EnqueueParams struct {
	EventType     string
	PayloadJSON   string
	NextAttemptAt time.Time
	ProcessingFor time.Duration
}

func (r *Repository) Enqueue(ctx context.Context, p EnqueueParams) error {
	const op = "outbox.repo.enqueue"

	const q = `
        INSERT INTO webhook_outbox (event_type, payload, status, attempts, next_attempt_at, processing_until, last_error, created_at, updated_at)
        VALUES ($1, $2::jsonb, 'pending', 0, $3, NULL, NULL, NOW(), NOW());
    `
	if _, err := r.exec.ExecContext(ctx, q, p.EventType, p.PayloadJSON, p.NextAttemptAt); err != nil {
		return dberrs.Map(err, op)
	}
	return nil
}

type Event struct {
	ID          int64  `db:"id"`
	EventType   string `db:"event_type"`
	PayloadJSON string `db:"payload"`
	Attempts    int    `db:"attempts"`
}

func (r *Repository) ClaimOne(ctx context.Context, processingFor time.Duration) (*Event, error) {
	const op = "outbox.repo.claim_one"

	const q = `
        WITH claimed AS (
            UPDATE webhook_outbox
            SET status = 'processing',
                attempts = attempts + 1,
                processing_until = NOW() + ($1::text)::interval,
                updated_at = NOW()
            WHERE id = (
                SELECT id
                FROM webhook_outbox
                WHERE
                    (status = 'pending' AND next_attempt_at <= NOW())
                    OR (status = 'processing' AND processing_until IS NOT NULL AND processing_until <= NOW())
                ORDER BY id
                FOR UPDATE SKIP LOCKED
                LIMIT 1
            )
            RETURNING id, event_type, payload, attempts
        )
        SELECT id, event_type, payload, attempts FROM claimed;
    `

	intervalText := processingFor.String()

	var ev Event
	if err := sqlx.GetContext(ctx, r.exec, &ev, q, intervalText); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, dberrs.Map(err, op)
	}
	return &ev, nil
}

func (r *Repository) MarkSucceeded(ctx context.Context, id int64) error {
	const op = "outbox.repo.mark_succeeded"

	const q = `
        UPDATE webhook_outbox
        SET status = 'succeeded',
            processing_until = NULL,
            updated_at = NOW()
        WHERE id = $1;
    `
	if _, err := r.exec.ExecContext(ctx, q, id); err != nil {
		return dberrs.Map(err, op)
	}
	return nil
}

func (r *Repository) MarkRetry(ctx context.Context, id int64, nextAttemptAt time.Time, lastErr string) error {
	const op = "outbox.repo.mark_retry"

	const q = `
        UPDATE webhook_outbox
        SET status = 'pending',
            next_attempt_at = $2,
            processing_until = NULL,
            last_error = $3,
            updated_at = NOW()
        WHERE id = $1;
    `
	if _, err := r.exec.ExecContext(ctx, q, id, nextAttemptAt, lastErr); err != nil {
		return dberrs.Map(err, op)
	}
	return nil
}

func (r *Repository) MarkDead(ctx context.Context, id int64, lastErr string) error {
	const op = "outbox.repo.mark_dead"

	const q = `
        UPDATE webhook_outbox
        SET status = 'dead',
            processing_until = NULL,
            last_error = $2,
            updated_at = NOW()
        WHERE id = $1;
    `
	if _, err := r.exec.ExecContext(ctx, q, id, lastErr); err != nil {
		return dberrs.Map(err, op)
	}
	return nil
}

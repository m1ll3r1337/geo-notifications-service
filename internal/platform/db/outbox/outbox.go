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

func (r *Repository) Enqueue(ctx context.Context, eventType string, payloadJSON string) error {
	const op = "outbox.repo.enqueue"

	const q = `
        INSERT INTO webhook_outbox (event_type, payload, status, attempts, next_attempt_at, created_at, updated_at)
        VALUES ($1, $2::jsonb, 'pending', 0, NOW(), NOW(), NOW());
    `
	if _, err := r.exec.ExecContext(ctx, q, eventType, payloadJSON); err != nil {
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

func (r *Repository) ClaimBatch(ctx context.Context, limit int) ([]Event, error) {
	const op = "outbox.repo.claim_batch"

	const q = `
        WITH claimed AS (
            SELECT id
            FROM webhook_outbox
            WHERE status = 'pending' AND next_attempt_at <= NOW()
            ORDER BY id
            FOR UPDATE SKIP LOCKED
            LIMIT $1
        )
        UPDATE webhook_outbox o
        SET attempts = attempts + 1,
            updated_at = NOW()
        FROM claimed
        WHERE o.id = claimed.id
        RETURNING o.id, o.event_type, o.payload, o.attempts;
    `

	var rows []Event
	if err := sqlx.SelectContext(ctx, r.exec, &rows, q, limit); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, dberrs.Map(err, op)
	}
	return rows, nil
}

func (r *Repository) MarkDispatched(ctx context.Context, id int64) error {
	const op = "outbox.repo.mark_dispatched"

	const q = `
        UPDATE webhook_outbox
        SET status = 'dispatched',
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
            last_error = $2,
            updated_at = NOW()
        WHERE id = $1;
    `
	if _, err := r.exec.ExecContext(ctx, q, id, lastErr); err != nil {
		return dberrs.Map(err, op)
	}
	return nil
}

package outboxrelay

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	outboxdb "github.com/m1ll3r1337/geo-notifications-service/internal/platform/db/outbox"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/db/uow"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/redis/queue"
)

type Logger interface {
	Info(ctx context.Context, msg string, args ...any)
	Error(ctx context.Context, msg string, args ...any)
}

type Relay struct {
	uow   *uow.UnitOfWork
	queue *queue.RedisQueue
	log   Logger

	batchSize     int
	pollInterval  time.Duration
	processingFor time.Duration
	maxAttempts   int
}

func New(db *sqlx.DB, q *queue.RedisQueue, log Logger) *Relay {
	return &Relay{
		uow:           uow.New(db),
		queue:         q,
		log:           log,
		batchSize:     100,
		pollInterval:  500 * time.Millisecond,
		processingFor: 30 * time.Second,
		maxAttempts:   10,
	}
}

func (r *Relay) Run(ctx context.Context) error {
	t := time.NewTicker(r.pollInterval)
	defer t.Stop()

	r.log.Info(ctx, "outbox relay started")
	for {
		select {
		case <-ctx.Done():
			r.log.Info(ctx, "outbox relay stopped")
			return ctx.Err()
		case <-t.C:
			if err := r.process(ctx); err != nil {
				r.log.Error(ctx, "outbox relay process failed", "error", err)
			}
		}
	}
}

func (r *Relay) process(ctx context.Context) error {
	var events []outboxdb.Event
	err := r.uow.WithinTxRoot(ctx, nil, func(sc uow.Scope) error {
		repo := outboxdb.New(sc.Executor())
		var err error
		events, err = repo.ClaimBatch(ctx, r.batchSize)
		return err
	})
	if err != nil || len(events) == 0 {
		return err
	}

	items := make([]queue.Item, 0, len(events))
	ids := make([]int64, 0, len(events))
	for _, ev := range events {
		items = append(items, queue.Item{
			EventType: ev.EventType,
			Payload:   ev.PayloadJSON,
			OutboxID:  ev.ID,
		})
		ids = append(ids, ev.ID)
	}

	pushErr := r.queue.EnqueueBatch(ctx, items)

	return r.uow.WithinTxRoot(ctx, nil, func(sc uow.Scope) error {
		repo := outboxdb.New(sc.Executor())

		if pushErr == nil {
			r.log.Info(ctx, "outbox relay dispatched batch", "count", len(ids))
			return repo.MarkDispatchedBatch(ctx, ids)
		}

		r.log.Error(ctx, "outbox relay enqueue failed", "error", pushErr, "count", len(ids))

		var retryIDs []int64
		for _, ev := range events {
			if ev.Attempts >= r.maxAttempts {
				_ = repo.MarkDead(ctx, ev.ID, pushErr.Error())
				continue
			}
			retryIDs = append(retryIDs, ev.ID)
		}

		next := time.Now().UTC().Add(backoff(minAttempts(events)))
		return repo.MarkRetryBatch(ctx, retryIDs, next, pushErr.Error())
	})
}

func backoff(attempt int) time.Duration {
	d := time.Duration(1<<attempt) * time.Second
	if d < 2*time.Second {
		return 2 * time.Second
	}
	if d > 5*time.Minute {
		return 5 * time.Minute
	}
	return d
}

func minAttempts(events []outboxdb.Event) int {
	if len(events) == 0 {
		return 1
	}
	min := events[0].Attempts
	for _, e := range events {
		if e.Attempts < min {
			min = e.Attempts
		}
	}
	if min < 1 {
		return 1
	}
	return min
}

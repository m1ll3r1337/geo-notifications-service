package webhookworker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/m1ll3r1337/geo-notifications-service/internal/domain/incidents"
)

type Logger interface {
	Info(ctx context.Context, msg string, args ...any)
	Error(ctx context.Context, msg string, args ...any)
}

type Worker struct {
	rdb      *redis.Client
	stream   string
	group    string
	consumer string

	httpClient *http.Client
	targetURL  string

	dedupeTTL time.Duration
	log       Logger
}

func New(rdb *redis.Client, stream, group, consumer, targetURL string, log Logger) *Worker {
	return &Worker{
		rdb:      rdb,
		stream:   stream,
		group:    group,
		consumer: consumer,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		targetURL: targetURL,
		dedupeTTL: 24 * time.Hour,
		log:       log,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	_ = w.rdb.XGroupCreateMkStream(ctx, w.stream, w.group, "0").Err()

	w.log.Info(ctx, "webhook worker started")
	for {
		select {
		case <-ctx.Done():
			w.log.Info(ctx, "webhook worker stopped")
			return ctx.Err()
		default:
		}

		streams, err := w.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    w.group,
			Consumer: w.consumer,
			Streams:  []string{w.stream, ">"},
			Count:    1,
			Block:    2 * time.Second,
		}).Result()
		if err == redis.Nil || err == nil && len(streams) == 0 {
			continue
		}
		if err != nil {
			w.log.Error(ctx, "redis read failed", "error", err)
			continue
		}

		for _, st := range streams {
			for _, msg := range st.Messages {
				if err := w.handle(ctx, msg.Values); err == nil {
					_ = w.rdb.XAck(ctx, w.stream, w.group, msg.ID).Err()
				} else {
					w.log.Error(ctx, "webhook handle failed", "error", err)
				}
			}
		}
	}
}

func (w *Worker) handle(ctx context.Context, values map[string]any) error {
	body, ok := values["body"].(string)
	if !ok {
		return fmt.Errorf("missing body")
	}

	outboxIDStr, ok := values["outbox_id"].(string)
	if !ok {
		return fmt.Errorf("missing outbox_id")
	}
	outboxID, err := strconv.ParseInt(outboxIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid outbox_id")
	}

	dedupeKey := fmt.Sprintf("processed:%d", outboxID)
	okNX, err := w.rdb.SetNX(ctx, dedupeKey, "1", w.dedupeTTL).Result()
	if err != nil {
		return err
	}
	if !okNX {
		return nil
	}

	var ev incidents.CheckCompleted
	if err := json.Unmarshal([]byte(body), &ev); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.targetURL, bytes.NewBufferString(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Event-Type", "location_check")
	req.Header.Set("Idempotency-Key", fmt.Sprintf("%d", ev.CheckID))

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook non-2xx: %d", resp.StatusCode)
	}

	w.log.Info(ctx, "webhook sent", "check_id", ev.CheckID, "outbox_id", outboxID)
	return nil
}

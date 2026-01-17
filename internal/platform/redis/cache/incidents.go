package incidentscache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	incidentsapp "github.com/m1ll3r1337/geo-notifications-service/internal/app/incidents"
	"github.com/m1ll3r1337/geo-notifications-service/internal/domain/incidents"
)

type Logger interface {
	Info(ctx context.Context, msg string, args ...any)
	Error(ctx context.Context, msg string, args ...any)
}

type Option func(*CachedRepository)

func WithTTL(ttl time.Duration) Option {
	return func(c *CachedRepository) { c.ttl = ttl }
}

func WithLogger(log Logger) Option {
	return func(c *CachedRepository) { c.log = log }
}

type CachedRepository struct {
	next incidentsapp.IncidentsRepository
	rdb  *redis.Client
	ttl  time.Duration
	log  Logger
}

func New(rdb *redis.Client, next incidentsapp.IncidentsRepository, opts ...Option) *CachedRepository {
	c := &CachedRepository{
		next: next,
		rdb:  rdb,
		ttl:  60 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *CachedRepository) Create(ctx context.Context, in incidents.CreateIncident) (incidents.Incident, error) {
	inc, err := c.next.Create(ctx, in)
	if err == nil {
		_ = c.bumpVersion(ctx)
	}
	return inc, err
}

func (c *CachedRepository) GetByID(ctx context.Context, id int64) (incidents.Incident, error) {
	return c.next.GetByID(ctx, id)
}

func (c *CachedRepository) List(ctx context.Context, f incidents.ListFilter) ([]incidents.Incident, error) {
	if !f.ActiveOnly {
		return c.next.List(ctx, f)
	}

	ver := c.getVersion(ctx)
	key := fmt.Sprintf("incidents:active:v%s:limit:%d:offset:%d", ver, f.Limit, f.Offset)

	if b, err := c.rdb.Get(ctx, key).Bytes(); err == nil {
		var cached []incidents.Incident
		if err := json.Unmarshal(b, &cached); err == nil {
			return cached, nil
		}
	}

	items, err := c.next.List(ctx, f)
	if err != nil {
		return nil, err
	}

	if b, err := json.Marshal(items); err == nil {
		if err := c.rdb.Set(ctx, key, b, c.ttl).Err(); err != nil && c.log != nil {
			c.log.Error(ctx, "incidents cache set failed", "error", err)
		}
	}

	return items, nil
}

func (c *CachedRepository) Update(ctx context.Context, id int64, in incidents.UpdateIncident) (incidents.Incident, error) {
	inc, err := c.next.Update(ctx, id, in)
	if err == nil {
		_ = c.bumpVersion(ctx)
	}
	return inc, err
}

func (c *CachedRepository) Deactivate(ctx context.Context, id int64) error {
	err := c.next.Deactivate(ctx, id)
	if err == nil {
		_ = c.bumpVersion(ctx)
	}
	return err
}

func (c *CachedRepository) FindNearby(ctx context.Context, p incidents.Point, limit int) ([]incidents.NearbyIncident, error) {
	return c.next.FindNearby(ctx, p, limit)
}

func (c *CachedRepository) CountUniqueUsersSince(ctx context.Context, since time.Time) (int, error) {
	return c.next.CountUniqueUsersSince(ctx, since)
}

func (c *CachedRepository) getVersion(ctx context.Context) string {
	const key = "incidents:active:version"
	val, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		_ = c.rdb.SetNX(ctx, key, "1", 0).Err()
		return "1"
	}
	if err != nil {
		if c.log != nil {
			c.log.Error(ctx, "incidents cache version get failed", "error", err)
		}
		return "0"
	}
	return val
}

func (c *CachedRepository) bumpVersion(ctx context.Context) error {
	const key = "incidents:active:version"
	if err := c.rdb.Incr(ctx, key).Err(); err != nil && c.log != nil {
		c.log.Error(ctx, "incidents cache version bump failed", "error", err)
		return err
	}
	return nil
}

package locationsdb

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/m1ll3r1337/geo-notifications-service/internal/domain/locations"
	dberrs "github.com/m1ll3r1337/geo-notifications-service/internal/platform/db/errs"
)

type Repository struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Repository { return &Repository{db: db} }

func (r *Repository) FindNearby(ctx context.Context, p locations.Point, limit int) ([]locations.NearbyIncident, error) {
	const op = "location.repo.find_nearby"

	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	const q = `
        SELECT
            i.id AS incident_id,
            ST_Distance(i.center, ST_MakePoint($1, $2)::geography) AS distance_m
        FROM incidents i
        WHERE i.active = TRUE
          AND ST_DWithin(i.center, ST_MakePoint($1, $2)::geography, i.radius)
        ORDER BY distance_m ASC
        LIMIT $3;
    `

	rows := []struct {
		IncidentID int64   `db:"incident_id"`
		DistanceM  float64 `db:"distance_m"`
	}{}

	if err := sqlx.SelectContext(ctx, r.db, &rows, q, p.Lon, p.Lat, limit); err != nil {
		return nil, dberrs.Map(err, op)
	}

	out := make([]locations.NearbyIncident, 0, len(rows))
	for _, row := range rows {
		out = append(out, locations.NearbyIncident{
			IncidentID:     row.IncidentID,
			DistanceMeters: row.DistanceM,
		})
	}
	return out, nil
}

func (r *Repository) RecordCheck(ctx context.Context, userID string, p locations.Point, incidentIDs []int64) (err error) {
	const op = "location.repo.record_check"

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return dberrs.Map(err, op)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			err = dberrs.Map(commitErr, op)
		}
	}()

	var checkID int64
	const qCheck = `
        INSERT INTO location_checks (user_id, location)
        VALUES ($1, ST_MakePoint($2, $3)::geography)
        RETURNING id;
    `
	if err := tx.QueryRowxContext(ctx, qCheck, userID, p.Lon, p.Lat).Scan(&checkID); err != nil {
		return dberrs.Map(err, op)
	}

	if len(incidentIDs) > 0 {
		const qLink = `INSERT INTO location_check_incidents (check_id, incident_id) VALUES ($1, $2);`
		for _, id := range incidentIDs {
			if _, err := tx.ExecContext(ctx, qLink, checkID, id); err != nil {
				return dberrs.Map(err, op)
			}
		}
	}
	//TODO: worker outbox
	return nil
}

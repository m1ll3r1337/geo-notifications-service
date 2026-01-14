package locationsdb

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/m1ll3r1337/geo-notifications-service/internal/domain/locations"
	dberrs "github.com/m1ll3r1337/geo-notifications-service/internal/platform/db/errs"
)

type Repository struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Repository { return &Repository{db: db} }

type dbNearbyIncident struct {
	IncidentID  int64     `db:"incident_id"`
	Title       string    `db:"title"`
	Description string    `db:"description"`
	Radius      int       `db:"radius"`
	CenterLon   float64   `db:"center_lon"`
	CenterLat   float64   `db:"center_lat"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
	DistanceM   float64   `db:"distance_m"`
}

func (d dbNearbyIncident) toDomain() locations.NearbyIncident {
	return locations.NearbyIncident{
		IncidentID:     d.IncidentID,
		DistanceMeters: d.DistanceM,
		Title:          d.Title,
		Description:    d.Description,
		Center:         locations.Point{Lat: d.CenterLat, Lon: d.CenterLon},
		Radius:         d.Radius,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
}

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
            i.title AS title,
            i.description AS description,
            i.radius AS radius,
            ST_X(i.center::geometry) AS center_lon,
            ST_Y(i.center::geometry) AS center_lat,
            i.created_at AS created_at,
            i.updated_at AS updated_at,
            ST_Distance(i.center, ST_MakePoint($1, $2)::geography) AS distance_m
        FROM incidents i
        WHERE i.active = TRUE
          AND ST_DWithin(i.center, ST_MakePoint($1, $2)::geography, i.radius)
        ORDER BY distance_m ASC
        LIMIT $3;
    `

	var rows []dbNearbyIncident
	if err := sqlx.SelectContext(ctx, r.db, &rows, q, p.Lon, p.Lat, limit); err != nil {
		return nil, dberrs.Map(err, op)
	}

	out := make([]locations.NearbyIncident, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toDomain())
	}
	return out, nil
}

func (r *Repository) RecordCheck(ctx context.Context, userID string, p locations.Point, incidentIDs []int64) error {
	const op = "locations.repo.record_check"

	var checkID int64
	const qCheck = `
        INSERT INTO location_checks (user_id, location)
        VALUES ($1, ST_MakePoint($2, $3)::geography)
        RETURNING id;
    `
	if err := sqlx.GetContext(ctx, r.db, &checkID, qCheck, userID, p.Lon, p.Lat); err != nil {
		return dberrs.Map(err, op)
	}

	if len(incidentIDs) > 0 {
		const qLink = `INSERT INTO location_check_incidents (check_id, incident_id) VALUES ($1, $2);`
		for _, id := range incidentIDs {
			if _, err := r.db.ExecContext(ctx, qLink, checkID, id); err != nil {
				return dberrs.Map(err, op)
			}
		}
	}

	return nil
}

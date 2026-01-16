package incidentsdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/m1ll3r1337/geo-notifications-service/internal/domain/incidents"
	dberrs "github.com/m1ll3r1337/geo-notifications-service/internal/platform/db/errs"
)

type Repository struct {
	exec sqlx.ExtContext
}

func New(db sqlx.ExtContext) *Repository { return &Repository{exec: db} }

type dbIncident struct {
	ID          int64          `db:"id"`
	Title       string         `db:"title"`
	Description sql.NullString `db:"description"`
	CenterLat   float64        `db:"center_lat"`
	CenterLon   float64        `db:"center_lon"`
	Radius      int            `db:"radius"`
	Active      bool           `db:"active"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
}

func (d dbIncident) toDomain() incidents.Incident {
	out := incidents.Incident{
		ID:    d.ID,
		Title: d.Title,
		Center: incidents.Point{
			Lat: d.CenterLat,
			Lon: d.CenterLon,
		},
		Radius:    d.Radius,
		Active:    d.Active,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
	if d.Description.Valid {
		out.Description = d.Description.String
	}
	return out
}

const selectIncidentCols = `
    id,
    title,
    description,
    ST_Y(center::geometry) AS center_lat,
    ST_X(center::geometry) AS center_lon,
    radius,
    active,
    created_at,
    updated_at
`

func (r *Repository) Create(ctx context.Context, in incidents.CreateIncident) (incidents.Incident, error) {
	const op = "incidents.repo.create"

	const q = `
        INSERT INTO incidents (title, description, center, radius)
        VALUES ($1, $2, ST_MakePoint($3, $4)::geography, $5)
        RETURNING ` + selectIncidentCols + `;
    `

	var row dbIncident
	if err := sqlx.GetContext(ctx, r.exec, &row, q,
		in.Title,
		nullString(in.Description),
		in.Center.Lon,
		in.Center.Lat,
		in.Radius,
	); err != nil {
		return incidents.Incident{}, dberrs.Map(err, op)
	}

	return row.toDomain(), nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (incidents.Incident, error) {
	const op = "incidents.repo.get_by_id"

	const q = `
        SELECT ` + selectIncidentCols + `
        FROM incidents
        WHERE id = $1;
    `

	var row dbIncident
	if err := sqlx.GetContext(ctx, r.exec, &row, q, id); err != nil {
		return incidents.Incident{}, dberrs.Map(err, op)
	}

	return row.toDomain(), nil
}

func (r *Repository) List(ctx context.Context, f incidents.ListFilter) ([]incidents.Incident, error) {
	const op = "incidents.repo.list"
	var (
		sb   strings.Builder
		args []any
	)
	limit := f.Limit
	offset := f.Offset

	sb.WriteString(`SELECT `)
	sb.WriteString(selectIncidentCols)
	sb.WriteString(` FROM incidents`)

	if f.ActiveOnly {
		sb.WriteString(` WHERE active = TRUE`)
	}

	sb.WriteString(` ORDER BY created_at DESC, id DESC LIMIT $`)
	args = append(args, limit)
	sb.WriteString(fmt.Sprintf("%d", len(args)))

	sb.WriteString(` OFFSET $`)
	args = append(args, offset)
	sb.WriteString(fmt.Sprintf("%d", len(args)))

	var rows []dbIncident
	if err := sqlx.SelectContext(ctx, r.exec, &rows, sb.String(), args...); err != nil {
		return nil, dberrs.Map(err, op)
	}

	out := make([]incidents.Incident, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.toDomain())
	}
	return out, nil
}

func (r *Repository) Deactivate(ctx context.Context, id int64) error {
	const op = "incidents.repo.deactivate"

	const q = `
        UPDATE incidents
        SET active = FALSE, updated_at = NOW()
        WHERE id = $1 AND active = TRUE
        RETURNING id;
    `

	var tmp int64
	if err := sqlx.GetContext(ctx, r.exec, &tmp, q, id); err != nil {
		return dberrs.Map(err, op)
	}
	return nil
}

func (r *Repository) Update(ctx context.Context, id int64, in incidents.UpdateIncident) (incidents.Incident, error) {
	const op = "incidents.repo.update"

	setParts := make([]string, 0, 6)
	args := make([]any, 0, 8)

	add := func(sqlPart string, val any) {
		args = append(args, val)
		setParts = append(setParts, fmt.Sprintf(sqlPart, len(args)))
	}

	if in.Title != nil {
		add("title = $%d", *in.Title)
	}
	if in.Description != nil {
		add("description = $%d", nullString(*in.Description))
	}
	if in.Center != nil {
		args = append(args, in.Center.Lon)
		lonPos := len(args)
		args = append(args, in.Center.Lat)
		latPos := len(args)
		setParts = append(setParts, fmt.Sprintf("center = ST_MakePoint($%d, $%d)::geography", lonPos, latPos))
	}
	if in.Radius != nil {
		add("radius = $%d", *in.Radius)
	}

	if len(setParts) == 0 {
		return r.GetByID(ctx, id)
	}

	setParts = append(setParts, "updated_at = NOW()")

	args = append(args, id)
	idPos := len(args)

	q := fmt.Sprintf(`
        UPDATE incidents
        SET %s
        WHERE id = $%d
        RETURNING %s;
    `, strings.Join(setParts, ", "), idPos, selectIncidentCols)

	var row dbIncident
	if err := sqlx.GetContext(ctx, r.exec, &row, q, args...); err != nil {
		return incidents.Incident{}, dberrs.Map(err, op)
	}

	return row.toDomain(), nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

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

func (d dbNearbyIncident) toDomain() incidents.NearbyIncident {
	return incidents.NearbyIncident{
		IncidentID:     d.IncidentID,
		DistanceMeters: d.DistanceM,
		Title:          d.Title,
		Description:    d.Description,
		Center:         incidents.Point{Lat: d.CenterLat, Lon: d.CenterLon},
		Radius:         d.Radius,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
	}
}

func (r *Repository) FindNearby(ctx context.Context, p incidents.Point, limit int) ([]incidents.NearbyIncident, error) {
	const op = "incidents.repo.find_nearby"
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
	if err := sqlx.SelectContext(ctx, r.exec, &rows, q, p.Lon, p.Lat, limit); err != nil {
		return nil, dberrs.Map(err, op)
	}

	out := make([]incidents.NearbyIncident, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toDomain())
	}
	return out, nil
}

func (r *Repository) RecordCheck(ctx context.Context, userID string, p incidents.Point, incidentIDs []int64) (int64, error) {
	const op = "incidents.repo.record_check"

	var checkID int64
	const qCheck = `
        INSERT INTO location_checks (user_id, location)
        VALUES ($1, ST_MakePoint($2, $3)::geography)
        RETURNING id;
    `
	if err := sqlx.GetContext(ctx, r.exec, &checkID, qCheck, userID, p.Lon, p.Lat); err != nil {
		return 0, dberrs.Map(err, op)
	}

	if len(incidentIDs) > 0 {
		const qLinkBatch = `
            INSERT INTO location_check_incidents (check_id, incident_id)
            SELECT $1, unnest($2::bigint[]);
        `
		if _, err := r.exec.ExecContext(ctx, qLinkBatch, checkID, incidentIDs); err != nil {
			return 0, dberrs.Map(err, op)
		}
	}

	return checkID, nil
}

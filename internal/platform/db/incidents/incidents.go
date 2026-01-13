package incidentsdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/m1ll3r1337/geo-notifications-service/internal/domain/incidents"
	dberrs "github.com/m1ll3r1337/geo-notifications-service/internal/platform/db/errs"
)

type Repository struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *Repository { return &Repository{db: db} }

type dbIncident struct {
	ID          int64          `db:"id"`
	Title       string         `db:"title"`
	Description sql.NullString `db:"description"`
	CenterLat   float64        `db:"center_lat"`
	CenterLon   float64        `db:"center_lon"`
	Radius      int            `db:"radius"`
	Active      bool           `db:"active"`
	CreatedAt   sql.NullTime   `db:"created_at"`
	UpdatedAt   sql.NullTime   `db:"updated_at"`
}

func (d dbIncident) toDomain() incidents.Incident {
	out := incidents.Incident{
		ID:    d.ID,
		Title: d.Title,
		Center: incidents.Point{
			Lat: d.CenterLat,
			Lon: d.CenterLon,
		},
		Radius: d.Radius,
		Active: d.Active,
	}
	if d.Description.Valid {
		out.Description = d.Description.String
	}
	if d.CreatedAt.Valid {
		out.CreatedAt = d.CreatedAt.Time
	}
	if d.UpdatedAt.Valid {
		out.UpdatedAt = d.UpdatedAt.Time
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
	if err := r.db.GetContext(ctx, &row, q,
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
	if err := r.db.GetContext(ctx, &row, q, id); err != nil {
		return incidents.Incident{}, dberrs.Map(err, op)
	}

	return row.toDomain(), nil
}

func (r *Repository) List(ctx context.Context, f incidents.ListFilter) ([]incidents.Incident, error) {
	const op = "incidents.repo.list"

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	var (
		sb   strings.Builder
		args []any
	)

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
	if err := r.db.SelectContext(ctx, &rows, sb.String(), args...); err != nil {
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
	if err := r.db.GetContext(ctx, &tmp, q, id); err != nil {
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
	if err := r.db.GetContext(ctx, &row, q, args...); err != nil {
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

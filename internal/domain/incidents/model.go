// Package incidents provides domain models and business logic for incident management.
package incidents

import (
	"strings"
	"time"

	"github.com/m1ll3r1337/geo-notifications-service/internal/errs"
)

type Point struct {
	Lat float64
	Lon float64
}

func (p Point) Validate(op string) error {
	fields := map[string]string{}

	if p.Lat < -90 || p.Lat > 90 {
		fields["lat"] = "must be between -90 and 90"
	}
	if p.Lon < -180 || p.Lon > 180 {
		fields["lon"] = "must be between -180 and 180"
	}
	if len(fields) > 0 {
		return errs.E(errs.KindInvalid, "INVALID_COORDINATES", op, "invalid coordinates", fields, nil)
	}

	return nil
}

type Incident struct {
	ID          int64
	Title       string
	Description string

	Center Point
	Radius int // meters

	Active bool

	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateIncident struct {
	Title       string
	Description string
	Center      Point
	Radius      int
}

func (c CreateIncident) Validate() error {
	const op = "incidents.model.validate_create"

	fields := map[string]string{}

	if strings.TrimSpace(c.Title) == "" {
		fields["title"] = "is required"
	}
	if c.Radius <= 0 {
		fields["radius"] = "must be > 0"
	}
	if err := c.Center.Validate(op); err != nil {
		return err
	}

	if len(fields) > 0 {
		return errs.E(errs.KindInvalid, "INCIDENT_INVALID", op, "invalid incident", fields, nil)
	}
	return nil
}

type UpdateIncident struct {
	Title       *string
	Description *string
	Center      *Point
	Radius      *int
}

func (u UpdateIncident) Validate() error {
	const op = "incidents.model.validate_update"

	fields := map[string]string{}

	if u.Title != nil && strings.TrimSpace(*u.Title) == "" {
		fields["title"] = "must not be empty"
	}
	if u.Radius != nil && *u.Radius <= 0 {
		fields["radius"] = "must be > 0"
	}
	if u.Center != nil {
		if err := u.Center.Validate(op); err != nil {
			return err
		}
	}

	if len(fields) > 0 {
		return errs.E(errs.KindInvalid, "INCIDENT_INVALID", op, "invalid incident", fields, nil)
	}
	return nil
}

type ListFilter struct {
	Limit      int
	Offset     int
	ActiveOnly bool
}

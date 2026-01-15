package incidents

import (
	"strings"
	"time"

	"github.com/m1ll3r1337/geo-notifications-service/internal/errs"
)

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

type NearbyIncident struct {
	IncidentID     int64
	DistanceMeters float64
	Title          string
	Description    string

	Center Point
	Radius int // meters

	CreatedAt time.Time
	UpdatedAt time.Time
}

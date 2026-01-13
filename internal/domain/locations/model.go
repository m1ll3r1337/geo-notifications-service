package locations

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

type CheckCommand struct {
	UserID string
	Point  Point
	Limit  int
}

func (c CheckCommand) Validate() error {
	const op = "location.model.validate_check"

	fields := map[string]string{}

	if strings.TrimSpace(c.UserID) == "" {
		fields["user_id"] = "is required"
	}
	if c.Limit < 0 {
		fields["limit"] = "must be >= 0"
	}
	if err := c.Point.Validate(op); err != nil {
		return err
	}

	if len(fields) > 0 {
		return errs.E(errs.KindInvalid, "CHECK_INVALID", op, "invalid request", fields, nil)
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

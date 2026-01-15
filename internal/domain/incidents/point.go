package incidents

import "github.com/m1ll3r1337/geo-notifications-service/internal/errs"

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

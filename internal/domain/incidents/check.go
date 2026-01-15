package incidents

import (
	"strings"

	"github.com/m1ll3r1337/geo-notifications-service/internal/errs"
)

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

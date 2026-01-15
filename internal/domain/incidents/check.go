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
	const op = "command.check.validate"

	if strings.TrimSpace(c.UserID) == "" {
		return errs.E(errs.KindInvalid, "INVALID_USER_ID", op, "user_id is required", map[string]string{"user_id": "is required"}, nil)
	}
	if err := c.Point.Validate(op); err != nil {
		return err
	}
	if c.Limit < 0 {
		return errs.E(errs.KindInvalid, "INVALID_LIMIT", op, "limit must be >= 0", map[string]string{"limit": "must be >= 0"}, nil)
	}
	if c.Limit > 500 {
		return errs.E(errs.KindInvalid, "INVALID_LIMIT", op, "limit must be <= 500", map[string]string{"limit": "must be <= 500"}, nil)
	}
	return nil
}

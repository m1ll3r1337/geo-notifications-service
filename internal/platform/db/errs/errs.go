package dberrs

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/m1ll3r1337/geo-notifications-service/internal/errs"
)

func Map(err error, op string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, sql.ErrNoRows) {
		return errs.E(errs.KindNotFound, "NOT_FOUND", op, "", nil, err)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.UniqueViolation:
			return errs.E(errs.KindConflict, "UNIQUE_VIOLATION", op, "", nil, err)
		case pgerrcode.ForeignKeyViolation:
			return errs.E(errs.KindConflict, "FK_VIOLATION", op, "", nil, err)
		case pgerrcode.CheckViolation:
			return errs.E(errs.KindInvalid, "CHECK_VIOLATION", op, "invalid input", nil, err)
		case pgerrcode.InvalidTextRepresentation:
			return errs.E(errs.KindInvalid, "INVALID_INPUT", op, "invalid input", nil, err)
		case pgerrcode.StringDataRightTruncationDataException:
			return errs.E(errs.KindInvalid, "VALUE_TOO_LONG", op, "value too long", nil, err)
		default:
			return errs.E(errs.KindInternal, "DB_ERROR", op, "database error", nil, err)
		}
	}

	return errs.E(errs.KindInternal, "INTERNAL", op, "internal error", nil, err)
}

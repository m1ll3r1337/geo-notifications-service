package healthdb

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/db"
)

type PostgresPinger struct {
	db *sqlx.DB
}

func NewPostgresPinger(db *sqlx.DB) PostgresPinger {
	return PostgresPinger{db: db}
}

func (p PostgresPinger) Ping(ctx context.Context) error {
	return db.StatusCheck(ctx, p.db)
}

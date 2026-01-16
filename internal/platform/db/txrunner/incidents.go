package txrunner

import (
	"context"
	"database/sql"

	incidentsapp "github.com/m1ll3r1337/geo-notifications-service/internal/app/incidents"
	incidentsdb "github.com/m1ll3r1337/geo-notifications-service/internal/platform/db/incidents"
	outboxdb "github.com/m1ll3r1337/geo-notifications-service/internal/platform/db/outbox"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/db/uow"
)

type IncidentsTxRunner struct {
	u *uow.UnitOfWork
}

func NewIncidentsTxRunner(u *uow.UnitOfWork) *IncidentsTxRunner {
	return &IncidentsTxRunner{u: u}
}

func (r *IncidentsTxRunner) WithinTx(
	ctx context.Context,
	fn func(ctx context.Context, checker incidentsapp.Checker, outbox incidentsapp.OutboxRepository) error) error {
	return r.u.WithinTxRoot(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted}, func(sc uow.Scope) error {
		locRepo := incidentsdb.New(sc.Executor())
		obRepo := outboxdb.New(sc.Executor())

		outboxWriter := outboxWriterAdapter{repo: obRepo}
		return fn(ctx, locRepo, outboxWriter)
	})
}

type outboxWriterAdapter struct {
	repo *outboxdb.Repository
}

func (a outboxWriterAdapter) Enqueue(ctx context.Context, eventType string, payloadJSON string) error {
	return a.repo.Enqueue(ctx, eventType, payloadJSON)
}

package uow

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type UnitOfWork struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *UnitOfWork { return &UnitOfWork{db: db} }

type Scope struct {
	exec sqlx.ExtContext
}

func (s Scope) Executor() sqlx.ExtContext { return s.exec }

func (s Scope) InTx() bool {
	_, ok := s.exec.(*sqlx.Tx)
	return ok
}

func (u *UnitOfWork) Scope() Scope { return Scope{exec: u.db} }

func (u *UnitOfWork) WithinTxRoot(ctx context.Context, opts *sql.TxOptions, fn func(Scope) error) error {
	return u.WithinTx(ctx, u.Scope(), opts, fn)
}

func (u *UnitOfWork) WithinTx(ctx context.Context, scope Scope, opts *sql.TxOptions, fn func(Scope) error) (err error) {
	if scope.InTx() {
		return fn(scope)
	}

	tx, err := u.db.BeginTxx(ctx, opts)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	txScope := Scope{exec: tx}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
			return
		}
		if commitErr := tx.Commit(); commitErr != nil {
			err = fmt.Errorf("commit tx: %w", commitErr)
		}
	}()

	return fn(txScope)
}

package pg

import (
	"context"
	"database/sql"
	"errors"
)

// NOTE: Most of this shamelessly lifted from ozzo-dbx,
// we don't need that whole pkg and even if we did the
// Tx implementation doesn't allow grabbing the *sql.Tx
// to integrate with watermill sql adapters.

type Beginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// TxContext starts a transaction and executes the given function with the given context and transaction options.
// If the function returns an error, the transaction will be rolled back.
// Otherwise, the transaction will be committed.
func TxContext(ctx context.Context, db Beginner, opts *sql.TxOptions, f func(*sql.Tx) error) (err error) {
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			if err2 := tx.Rollback(); err2 != nil {
				if err2 == sql.ErrTxDone {
					return
				}
				err = errors.Join(err, err2)
			}
		} else {
			if err = tx.Commit(); err == sql.ErrTxDone {
				err = nil
			}
		}
	}()

	err = f(tx)

	return err
}

# PostgreSQL

`Connector` owns its database handle and closes it with `Close`. A
`TransactionalClient` borrows that handle; it must not be closed by callers.

`ConnectionProvider.Connection(ctx)` acquires a distinct `*sql.Conn` for every
call. Context identity never controls sharing. The caller owns the returned
connection and must call `Close`, normally with `defer` immediately after a
successful acquisition. `Close` is idempotent and releases the underlying
connection at most once.

Start a transaction with `BeginTransaction(ctx, opts)`. The supplied context is
used for creation and `opts` is passed to `database/sql`. An acquired connection
allows one active transaction. Starting another transaction before the first is
committed or rolled back returns `ErrNestedTransaction`; transactions are not
shared across nested or concurrent work. Finish each transaction with exactly
one `Commit` or `Rollback` before closing its connection.

When composing a connection and transaction around application work, defer
connection closure and roll back on an operation error. If cleanup can fail,
use `errors.Join` to retain both the business and cleanup errors.

## Transaction example

The caller supplies the operation context, owns both cleanup paths, and keeps
the business error when cleanup also fails:

```go
func createUser(ctx context.Context, provider postgresql.ConnectionProvider, email string) (err error) {
	conn, err := provider.Connection(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, conn.Close())
	}()

	tx, err := conn.BeginTransaction(ctx, nil)
	if err != nil {
		return err
	}
	finished := false
	defer func() {
		if !finished {
			err = errors.Join(err, tx.Rollback())
		}
	}()

	if _, err := tx.ExecContext(ctx, `INSERT INTO users (email) VALUES ($1)`, email); err != nil {
		return err
	}

	finished = true // Commit is terminal even when it reports an error.
	return tx.Commit()
}
```

# PostgreSQL

`Connector` owns its database handle and closes it with `Close`. A
`TransactionalClient` borrows that handle; it must not be closed by callers.
`Connector` also implements the narrow `Pinger` interface through
`Ping(ctx)`, which is suitable for readiness checks without exposing query or
transaction operations.

`Connector` has no usable zero value; construct it with `NewConnector`. Do not
call `Open`, `Close`, or create derived clients concurrently on the same
connector. `Close` stops new work and waits for in-use database connections as
defined by `database/sql`; callers should stop their HTTP server and background
work before closing it. `TransactionalClient` is safe to share after a
successful `Open`, subject to the normal `database/sql` guarantees.

Open a connector with `Open(ctx, dsn, config)`. It configures the pool and
calls `PingContext` before returning, so a successful call represents a usable
database connection rather than only a parsed DSN. `ConnectTimeout` bounds
that initial ping when positive; zero adds no deadline beyond the caller's
context and a negative value is rejected. If validation fails, the newly opened
handle is closed and the validation and close errors are joined.

All pool settings default to `database/sql` defaults when zero: unlimited open
connections, the driver's normal idle-connection default, and no maximum
connection or idle lifetime. Set positive `MaxOpenConnections`,
`MaxIdleConnections`, `ConnectionMaxLifetime`, and `ConnectionMaxIdleTime` to
override them. Negative settings are rejected. `MaxConnections` and
`ConnectionLifetime` remain compatibility aliases for the corresponding
explicit fields.

`Migrator.MigrateUp(ctx)` uses the caller's context for acquiring a connection,
waiting for its advisory lock, schema queries, and migration transactions.
Canceling the context therefore interrupts lock and connection waits. Migration
files must be regular files named
`<numeric-version>_<name>.up.sql`; versions are ordered numerically and two
filenames with the same numeric version (including differently zero-padded
forms) are rejected before any database mutation. An applied version whose
file is missing remains an error. Set `MigrationAdvisoryLockID` to a stable,
application-specific value when unrelated applications share a cluster; zero
uses Go Kita's legacy default lock ID. If the caller context is canceled after
the lock is acquired, cleanup uses a separate five-second context to release
the session lock; if release fails, the connection is discarded rather than
returned to the pool with a possibly held lock.

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

`DSN.String` builds a PostgreSQL URL from its fields. It does not validate that
the host, port, database, or credentials are present; connection errors are
reported by `Open`. `Migrator` is created from an open connector and borrows
its pool. It does not own the connector and must not outlive it. Run migrations
once during startup; concurrent calls coordinate only through the configured
database advisory lock.

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

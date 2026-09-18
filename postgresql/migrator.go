package postgresql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/go-faster/errors"
	"github.com/jmoiron/sqlx"

	"github.com/nightnoryu/go-kita/log"
)

const createSchemaMigrationTableSQL = `
CREATE TABLE IF NOT EXISTS schema_migration (
    version TEXT PRIMARY KEY,
    executed_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`

const (
	migrationAdvisoryLockID int64 = 0x00676f2d6b697461 // "go-kita"
	acquireMigrationLockSQL       = "SELECT pg_advisory_lock($1)"
	releaseMigrationLockSQL       = "SELECT pg_advisory_unlock($1)"
	migrationCleanupTimeout       = 5 * time.Second
)

var migrationFileRegexp = regexp.MustCompile(`^(\d+)_(.+)\.up\.sql$`)

type Migrator interface {
	// MigrateUp applies pending migrations using ctx for all database work,
	// including waiting for the advisory lock.
	MigrateUp(ctx context.Context) error
}

type migrator struct {
	db             *sqlx.DB
	logger         log.Logger
	fs             fs.FS
	advisoryLockID int64
}

type migrationFile struct {
	version string
	number  uint64
	name    string
	path    string
}

func (m migrator) MigrateUp(ctx context.Context) (err error) {
	if ctx == nil {
		return fmt.Errorf("postgresql: nil context")
	}

	conn, err := m.db.Connx(ctx)
	if err != nil {
		err = errors.Wrap(err, "failed to open migrator connection")
		return err
	}

	defer func() {
		closeErr := conn.Close()
		if closeErr != nil && !errors.Is(closeErr, sql.ErrConnDone) {
			err = errors.Join(err, closeErr)
		}
	}()

	if _, err = conn.ExecContext(ctx, acquireMigrationLockSQL, m.advisoryLockID); err != nil {
		return errors.Wrap(err, "failed to acquire migration advisory lock")
	}

	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), migrationCleanupTimeout)
		defer cancel()
		if _, unlockErr := conn.ExecContext(cleanupCtx, releaseMigrationLockSQL, m.advisoryLockID); unlockErr != nil {
			err = errors.Join(err, errors.Wrap(unlockErr, "failed to release migration advisory lock"))
			if discardErr := conn.Raw(func(any) error { return driver.ErrBadConn }); discardErr != nil {
				err = errors.Join(err, errors.Wrap(discardErr, "failed to discard connection after advisory lock release failure"))
			}
		}
	}()

	files, err := m.readMigrationFiles()
	if err != nil {
		return errors.Wrap(err, "failed to read migration files")
	}

	if _, err = conn.ExecContext(ctx, createSchemaMigrationTableSQL); err != nil {
		return errors.Wrap(err, "failed to create schema_migration table")
	}

	var appliedVersions []string
	if err = conn.SelectContext(ctx, &appliedVersions, "SELECT version FROM schema_migration"); err != nil {
		return errors.Wrap(err, "failed to read applied migrations")
	}

	fileVersions := make(map[string]struct{}, len(files))
	for _, f := range files {
		fileVersions[f.version] = struct{}{}
	}
	for _, version := range appliedVersions {
		if _, ok := fileVersions[version]; !ok {
			return fmt.Errorf("migration %s is applied but its file is missing from data/migrations", version)
		}
	}

	applied := make(map[string]struct{}, len(appliedVersions))
	for _, version := range appliedVersions {
		applied[version] = struct{}{}
	}

	for _, f := range files {
		if _, ok := applied[f.version]; ok {
			continue
		}
		if err = m.applyMigration(ctx, conn, f); err != nil {
			return errors.Wrapf(err, "failed to apply migration %s", f.version)
		}
	}

	return nil
}

func (m migrator) applyMigration(ctx context.Context, conn *sqlx.Conn, f migrationFile) error {
	content, err := fs.ReadFile(m.fs, f.path)
	if err != nil {
		return errors.Wrap(err, "failed to read migration file")
	}

	tx, err := conn.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}

	start := time.Now()

	if _, err = tx.ExecContext(ctx, string(content)); err != nil {
		execErr := errors.Wrap(err, "failed to execute migration")
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Join(execErr, errors.Wrap(rollbackErr, "failed to roll back migration"))
		}
		return execErr
	}

	if _, err = tx.ExecContext(ctx, "INSERT INTO schema_migration (version) VALUES ($1)", f.version); err != nil {
		recordErr := errors.Wrap(err, "failed to record migration version")
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Join(recordErr, errors.Wrap(rollbackErr, "failed to roll back migration"))
		}
		return recordErr
	}

	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit migration")
	}

	duration := time.Since(start)
	m.logger.WithFields(log.Fields{"version": f.version, "duration": duration}).Info("migration complete")

	return nil
}

func (m migrator) readMigrationFiles() ([]migrationFile, error) {
	entries, err := fs.ReadDir(m.fs, ".")
	if err != nil {
		return nil, err
	}

	files := make([]migrationFile, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("read migration entry %q: %w", entry.Name(), err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("migration entry %q is not a regular file", entry.Name())
		}
		match := migrationFileRegexp.FindStringSubmatch(entry.Name())
		if match == nil {
			return nil, fmt.Errorf("malformed migration filename %q: expected <version>_<name>.up.sql", entry.Name())
		}
		number, err := strconv.ParseUint(match[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("malformed migration version in filename %q: %w", entry.Name(), err)
		}
		files = append(files, migrationFile{
			version: match[1],
			number:  number,
			name:    match[2],
			path:    entry.Name(),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].number < files[j].number
	})
	for i := 1; i < len(files); i++ {
		if files[i-1].number == files[i].number {
			return nil, fmt.Errorf("duplicate migration version %d in %q and %q", files[i].number, files[i-1].path, files[i].path)
		}
	}

	return files, nil
}

package postgresql

import (
	"context"
	"database/sql"
	"io/fs"
	"regexp"
	"sort"
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

var migrationFileRegexp = regexp.MustCompile(`^(\d+)_(.+)\.up\.sql$`)

type Migrator interface {
	MigrateUp() error
}

type migrator struct {
	db     *sqlx.DB
	logger log.Logger
	fs     fs.FS
}

type migrationFile struct {
	version string
	name    string
	path    string
}

func (m migrator) MigrateUp() (err error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	if _, err = conn.ExecContext(ctx, createSchemaMigrationTableSQL); err != nil {
		return errors.Wrap(err, "failed to create schema_migration table")
	}

	files, err := m.readMigrationFiles()
	if err != nil {
		return errors.Wrap(err, "failed to read migration files")
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
			return errors.Wrapf(err, "migration %s is applied but its file is missing from data/migrations", version)
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
		_ = tx.Rollback()
		return errors.Wrap(err, "failed to execute migration: %w")
	}

	if _, err = tx.ExecContext(ctx, "INSERT INTO schema_migration (version) VALUES ($1)", f.version); err != nil {
		_ = tx.Rollback()
		return errors.Wrap(err, "failed to record migration version")
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
		if entry.IsDir() {
			continue
		}
		match := migrationFileRegexp.FindStringSubmatch(entry.Name())
		if match == nil {
			continue
		}
		files = append(files, migrationFile{
			version: match[1],
			name:    match[2],
			path:    entry.Name(),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].version < files[j].version
	})

	return files, nil
}

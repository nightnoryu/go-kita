package postgresql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestMigrator_ReadMigrationFilesOrdersVersionsNumerically(t *testing.T) {
	m := migrator{fs: migrationTestFS(t, "10_ten.up.sql", "2_two.up.sql")}

	files, err := m.readMigrationFiles()
	require.NoError(t, err)
	require.Len(t, files, 2)
	require.Equal(t, "2", files[0].version)
	require.Equal(t, "10", files[1].version)
}

func TestMigrator_ReadMigrationFilesRejectsDuplicateNumericVersions(t *testing.T) {
	m := migrator{fs: migrationTestFS(t, "001_initial.up.sql", "1_other.up.sql")}

	_, err := m.readMigrationFiles()
	require.ErrorContains(t, err, "duplicate migration version 1")
}

func TestMigrator_ReadMigrationFilesRejectsMalformedFilename(t *testing.T) {
	m := migrator{fs: migrationTestFS(t, "initial.sql")}

	_, err := m.readMigrationFiles()
	require.ErrorContains(t, err, "malformed migration filename")
}

func TestMigrator_ReadMigrationFilesRejectsNonRegularFile(t *testing.T) {
	dir := t.TempDir()
	realFile := filepath.Join(dir, "migration.sql")
	require.NoError(t, os.WriteFile(realFile, []byte("SELECT 1;"), 0o600))
	require.NoError(t, os.Symlink(realFile, filepath.Join(dir, "1_initial.up.sql")))

	m := migrator{fs: os.DirFS(dir)}
	_, err := m.readMigrationFiles()
	require.ErrorContains(t, err, "not a regular file")
}

func TestMigrator_ReleasesLockWithCleanupContextAfterCancellation(t *testing.T) {
	state := &migrationCleanupDriverState{}
	db := sqlx.NewDb(openMigrationCleanupTestDB(t, state), "migration-cleanup-test")
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	ctx, cancel := context.WithCancel(context.Background())
	m := migrator{db: db, fs: migrationTestFS(t, "bad.sql"), advisoryLockID: 1}
	state.cancel = cancel

	err := m.MigrateUp(ctx)
	require.Error(t, err)
	require.True(t, state.unlockCalled())
	require.False(t, state.unlockContextCanceled())
}

func TestValidateConfig(t *testing.T) {
	require.NoError(t, validateConfig(Config{}))
	require.Error(t, validateConfig(Config{ConnectTimeout: -1}))
	require.Error(t, validateConfig(Config{MaxIdleConnections: -1}))
}

func migrationTestFS(t *testing.T, names ...string) fs.FS {
	t.Helper()
	dir := t.TempDir()
	for _, name := range names {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte("SELECT 1;"), 0o600))
	}
	return os.DirFS(dir)
}

type migrationCleanupDriverState struct {
	mu                     sync.Mutex
	cancel                 context.CancelFunc
	unlocked               bool
	unlockContextWasCancel bool
}

func (s *migrationCleanupDriverState) unlockCalled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.unlocked
}

func (s *migrationCleanupDriverState) unlockContextCanceled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.unlockContextWasCancel
}

type migrationCleanupDriver struct{ state *migrationCleanupDriverState }

func (d migrationCleanupDriver) Open(string) (driver.Conn, error) {
	return migrationCleanupConn(d), nil
}

type migrationCleanupConn struct{ state *migrationCleanupDriverState }

func (c migrationCleanupConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (c migrationCleanupConn) Begin() (driver.Tx, error) { return nil, errors.New("not implemented") }
func (c migrationCleanupConn) Close() error              { return nil }
func (c migrationCleanupConn) ExecContext(ctx context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	switch query {
	case acquireMigrationLockSQL:
		c.state.cancel()
	case releaseMigrationLockSQL:
		c.state.unlocked = true
		c.state.unlockContextWasCancel = ctx.Err() != nil
	}
	return driver.RowsAffected(0), nil
}

var migrationCleanupDriverSequence int

func openMigrationCleanupTestDB(t *testing.T, state *migrationCleanupDriverState) *sql.DB {
	t.Helper()
	migrationCleanupDriverSequence++
	name := "go-kita-postgresql-migration-cleanup-test-" + strconv.Itoa(migrationCleanupDriverSequence)
	sql.Register(name, migrationCleanupDriver{state: state})
	db, err := sql.Open(name, "")
	require.NoError(t, err)
	return db
}

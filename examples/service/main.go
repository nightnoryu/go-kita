package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nightnoryu/go-kita/env"
	"github.com/nightnoryu/go-kita/health"
	"github.com/nightnoryu/go-kita/jsonlog"
	"github.com/nightnoryu/go-kita/log"
	"github.com/nightnoryu/go-kita/postgresql"
	"github.com/nightnoryu/go-kita/redis"
)

//go:embed migrations/*.up.sql
var migrationFiles embed.FS

type config struct {
	ListenAddress   string         `env:"LISTEN_ADDRESS" envDefault:":8080"`
	LogLevel        jsonlog.Level  `env:"LOG_LEVEL" envDefault:"info"`
	ShutdownTimeout time.Duration  `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	Postgres        postgresConfig `envPrefix:"POSTGRES_"`
	Redis           redisConfig    `envPrefix:"REDIS_"`
}

type postgresConfig struct {
	Host     string `env:"HOST,required"`
	Port     int    `env:"PORT" envDefault:"5432"`
	Database string `env:"DATABASE,required"`
	User     string `env:"USER,required"`
	Password string `env:"PASSWORD,required"`
}

type redisConfig struct {
	Host     string `env:"HOST,required"`
	Port     int    `env:"PORT" envDefault:"6379"`
	Password string `env:"PASSWORD"`
	DB       int    `env:"DB" envDefault:"0"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		panic(err)
	}
}

func run(ctx context.Context) (err error) {
	cfg, err := env.ParseEnv[config]("example-service")
	if err != nil {
		return err
	}

	logger, err := jsonlog.NewLogger(&jsonlog.Config{Level: cfg.LogLevel, AppName: "example-service"})
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	defer func() {
		// Zap writes to stderr. A terminal or pipe may not support fsync even
		// though every log entry was written successfully.
		syncErr := logger.Sync()
		if errors.Is(syncErr, syscall.EINVAL) || errors.Is(syncErr, syscall.ENOTSUP) {
			return
		}
		err = errors.Join(err, syncErr)
	}()

	database := postgresql.NewConnector()
	if err = database.Open(ctx, postgresql.DSN{
		Host: cfg.Postgres.Host, Port: cfg.Postgres.Port, Database: cfg.Postgres.Database,
		User: cfg.Postgres.User, Password: cfg.Postgres.Password,
	}, postgresql.Config{ConnectTimeout: 5 * time.Second}); err != nil {
		return err
	}
	defer func() { err = errors.Join(err, database.Close()) }()

	migrations, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}
	migrator, err := database.Migrator(logger, migrations)
	if err != nil {
		return err
	}
	if err = migrator.MigrateUp(ctx); err != nil {
		return err
	}

	cache, err := redis.OpenClient(ctx, redis.DSN{
		Host: cfg.Redis.Host, Port: cfg.Redis.Port, Password: cfg.Redis.Password, DB: cfg.Redis.DB,
	}, redis.Config{})
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, cache.Close()) }()

	server, err := newServer(cfg.ListenAddress, database, cache, logger)
	if err != nil {
		return err
	}

	logger.WithFields(log.Fields{"address": cfg.ListenAddress}).Info("service started")
	serverErr := make(chan error, 1)
	go func() { serverErr <- server.ListenAndServe() }()

	select {
	case err = <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	}
}

func newServer(address string, database postgresql.Connector, cache redis.Client, logger log.Logger) (*http.Server, error) {
	ready, err := health.NewReadinessHandler(health.ReadinessConfig{
		Timeout:      2 * time.Second,
		CheckTimeout: time.Second,
		Checks: []health.NamedCheck{
			{Name: "postgres", Check: database.Ping},
			{Name: "redis", Check: cache.Ping},
		},
		OnFailure: func(name string, err error) {
			logger.WithFields(log.Fields{"dependency": name}).Error(err, "readiness check failed")
		},
	})
	if err != nil {
		return nil, err
	}
	live, err := health.NewLivenessHandler(health.LivenessConfig{})
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle("GET /healthz", live)
	mux.Handle("GET /readyz", ready)
	return &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}, nil
}

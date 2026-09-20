package jsonlog

import (
	"errors"
	"fmt"
	"sort"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/nightnoryu/go-kita/log"
)

const (
	appNameKey = "app_name"
	timeKey    = "time"
)

type logger struct {
	*zap.Logger
}

func NewLogger(config *Config) (log.MainLogger, error) {
	if config == nil {
		return nil, errors.New("jsonlog: config is nil")
	}
	if !config.Level.valid() {
		return nil, fmt.Errorf("jsonlog: unsupported log level %d", config.Level)
	}

	implConfig := zap.NewProductionConfig()

	implConfig.Level = zap.NewAtomicLevelAt(zapcore.Level(config.Level))
	implConfig.DisableCaller = true

	implConfig.EncoderConfig.TimeKey = timeKey
	implConfig.EncoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder

	impl, err := implConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("build zap logger: %w", err)
	}
	return &logger{
		Logger: impl.With(zap.String(appNameKey, config.AppName)),
	}, nil
}

func (l *logger) WithFields(fields log.Fields) log.Logger {
	implFields := make([]zap.Field, 0, len(fields))
	for _, key := range buildOrderedFieldKeys(fields) {
		switch v := fields[key].(type) {
		case string:
			implFields = append(implFields, zap.String(key, v))
		case int:
			implFields = append(implFields, zap.Int(key, v))
		case bool:
			implFields = append(implFields, zap.Bool(key, v))
		default:
			implFields = append(implFields, zap.Any(key, fields[key]))
		}
	}
	return &logger{l.With(implFields...)}
}

func (l *logger) Debug(args ...any) {
	l.Logger.Debug(fmt.Sprint(args...))
}

func (l *logger) Info(args ...any) {
	l.Logger.Info(fmt.Sprint(args...))
}

func (l *logger) Warn(args ...any) {
	l.Logger.Warn(fmt.Sprint(args...))
}

func (l *logger) Error(err error, args ...any) {
	l.With(zap.Error(err)).Error(fmt.Sprint(args...))
}

func (l *logger) FatalError(err error, args ...any) {
	l.With(zap.Error(err)).Fatal(fmt.Sprint(args...))
}

func buildOrderedFieldKeys(fields log.Fields) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

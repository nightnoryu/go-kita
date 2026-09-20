package jsonlog

import (
	"fmt"
	"strings"

	"go.uber.org/zap/zapcore"
)

type Level zapcore.Level

const (
	FatalLevel = Level(zapcore.FatalLevel)
	ErrorLevel = Level(zapcore.ErrorLevel)
	WarnLevel  = Level(zapcore.WarnLevel)
	InfoLevel  = Level(zapcore.InfoLevel)
	DebugLevel = Level(zapcore.DebugLevel)

	debugLevelName   = "debug"
	infoLevelName    = "info"
	warnLevelName    = "warn"
	warningLevelName = "warning"
	errorLevelName   = "error"
	fatalLevelName   = "fatal"
)

func (l *Level) UnmarshalText(text []byte) error {
	var parsed Level
	switch strings.ToLower(strings.TrimSpace(string(text))) {
	case debugLevelName:
		parsed = DebugLevel
	case infoLevelName:
		parsed = InfoLevel
	case warnLevelName, warningLevelName:
		parsed = WarnLevel
	case errorLevelName:
		parsed = ErrorLevel
	case fatalLevelName:
		parsed = FatalLevel
	default:
		return fmt.Errorf(
			"unknown log level %q: use %s, %s, %s, %s, or %s",
			text,
			debugLevelName,
			infoLevelName,
			warnLevelName,
			errorLevelName,
			fatalLevelName,
		)
	}
	*l = parsed
	return nil
}

func (l Level) valid() bool {
	switch l {
	case DebugLevel, InfoLevel, WarnLevel, ErrorLevel, FatalLevel:
		return true
	default:
		return false
	}
}

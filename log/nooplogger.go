package log

// NoopLogger discards all log entries.
type NoopLogger struct{}

func (NoopLogger) WithFields(Fields) Logger { return NoopLogger{} }

func (NoopLogger) Debug(...any) {}

func (NoopLogger) Info(...any) {}

func (NoopLogger) Warn(...any) {}

func (NoopLogger) Error(error, ...any) {}

func (NoopLogger) FatalError(error, ...any) {}

func (NoopLogger) Sync() error { return nil }

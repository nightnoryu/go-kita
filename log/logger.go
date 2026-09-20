package log

type Fields map[string]any

// Logger - simplified logger abstraction
type Logger interface {
	WithFields(Fields) Logger
	Debug(...any)
	Info(...any)
	Warn(...any)
	Error(error, ...any)
}

// MainLogger - Logger which can also report fatal errors and handle buffer flushing
type MainLogger interface {
	Logger
	// FatalError prints out an error and calls os.Exit(1)
	FatalError(error, ...any)
	// Sync flushes any buffered records. Call this using defer
	Sync() error
}

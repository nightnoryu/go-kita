# log

Logging abstractions and helpers shared across services.

`NoopLogger` discards every entry and is useful in tests.

## Example

```go
package main

import (
	"github.com/nightnoryu/go-kita/jsonlog"
	"github.com/nightnoryu/go-kita/log"
)

func main() {
	logger, err := jsonlog.NewLogger(&jsonlog.Config{
		Level:   jsonlog.InfoLevel,
		AppName: "test",
	})
	if err != nil {
		panic(err)
	}
	defer func() { _ = logger.Sync() }()

	opts := log.DefaultTrimForLogsOpts
	opts.SensitiveFields = []string{"password", "secret"} // matched case-insensitively
	opts.SensitivePlaceholder = "HIDDEN"

	params := map[string]string{
		"not_sensitive": "test",
		"Password":      "correct horse battery staple",
	}

	logger.
		WithFields(log.Fields{"args": log.TrimForLogs(params, opts)}).
		Info("call finished")
}
```

`TrimForLogs` limits strings by bytes, not runes. Its default limits are 100 bytes per string and 10 elements per
slice; a non-positive limit disables the respective truncation.

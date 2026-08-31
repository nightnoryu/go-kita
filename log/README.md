# log

Logging abstractions and helpers shared across services.

## Example

```go
package main

import (
	"github.com/nightnoryu/go-kita/jsonlog"
	"github.com/nightnoryu/go-kita/log"
)

func main() {
	logger := jsonlog.NewLogger(&jsonlog.Config{
		Level:   jsonlog.InfoLevel,
		AppName: "test",
	})
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

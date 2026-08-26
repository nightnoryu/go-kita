# log

General logging utilities.

## TrimForLogs example

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
	
	options := log.DefaultTrimForLogsOpts
	options.SensitiveFields = []string{
		"Password",
		"password",
		"secret",
	}
	options.SensitivePlaceholder = "HIDDEN"

	params := map[string]string{
		"not_sensitive": "test",
		"password": "correct horse battery staple",
    }
	
	fields := log.Fields{
		"args": log.TrimForLogs(params, options),
	}

	loggerWithRequestFields := logger.WithFields(fields)
	loggerWithRequestFields.Info("call finished")
}
```

# jsonlog

Structured JSON logging.

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
	defer logger.Sync()

	fields := log.Fields{
		"duration": "1s",
		"method":   "GET /test",
	}

	loggerWithRequestFields := logger.WithFields(fields)
	loggerWithRequestFields.Info("call finished")

	logger.FatalError("fatal error")
}
```

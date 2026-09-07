<p align="center"><img src="https://github.com/user-attachments/assets/07235b5a-1a5e-467d-a7f9-a8662d6d0e24" width="160" title="Go Kita Logo"></p>

<p align="center">
  <a href="https://github.com/nightnoryu/go-kita/releases"><img src="https://img.shields.io/github/release/nightnoryu/go-kita.svg?cache-control=no-cache"></a>
  <a href="https://pkg.go.dev/github.com/nightnoryu/go-kita"><img src="https://pkg.go.dev/badge/github.com/nightnoryu/go-kita.svg?cache-control=no-cache"></a>
  <a href="https://github.com/nightnoryu/go-kita/blob/main/LICENSE"><img src="https://img.shields.io/github/license/nightnoryu/go-kita?cache-control=no-cache"></a>
  <a href="https://github.com/nightnoryu/go-kita/actions/workflows/ci.yml"><img src="https://github.com/nightnoryu/go-kita/actions/workflows/ci.yml/badge.svg?cache-control=no-cache"></a>
</p>

Go Kita is a lightweight SDK for building microservices without rewriting the same infrastructure boilerplate.

## 🎯 Why

I kept rewriting the same infrastructure code across my Go projects.
Go Kita is an attempt to extract those patterns into a small, reusable toolkit.

The project also serves as a playground for experimenting with microservice infrastructure.

> [!NOTE]
> Go Kita is primarily built for my own side projects and is not intended
> to be a general-purpose production framework.

## ✨ Features

- Structured JSON logging based on [zap](https://github.com/uber-go/zap)
- Maybe monad implementation with distinct Absent and None value semantics
- PostgreSQL connectivity with custom migrator and connection pool based on [pgx](https://github.com/jackc/pgx)
- Redis connectivity based on [go-redis](https://github.com/redis/go-redis)
- Environment variables parsing based on [caarlos0/env](https://github.com/caarlos0/env)
- Runtime helpers
- Generic slices functions

Check README in each package for more details.

## 🚀 Quick Start

```shell
go get github.com/nightnoryu/go-kita@latest
```

A small example that wires several packages together — config from the
environment, structured logging, graceful shutdown, and a slice helper:

```go
package main

import (
	"context"

	"github.com/nightnoryu/go-kita/env"
	"github.com/nightnoryu/go-kita/jsonlog"
	"github.com/nightnoryu/go-kita/log"
	"github.com/nightnoryu/go-kita/runtime"
	"github.com/nightnoryu/go-kita/slices"
)

type config struct {
	Debug   bool `env:"DEBUG" envDefault:"false"`
	Workers int  `env:"WORKERS" envDefault:"4"`
}

func main() {
	// Reads MYAPP_DEBUG, MYAPP_WORKERS.
	cfg, err := env.ParseEnv[config]("myapp")
	if err != nil {
		panic(err)
	}

	level := jsonlog.InfoLevel
	if cfg.Debug {
		level = jsonlog.DebugLevel
	}

	logger := jsonlog.NewLogger(&jsonlog.Config{
		Level:   level,
		AppName: "myapp",
	})
	defer func() { _ = logger.Sync() }()

	// Canceled on SIGINT / SIGTERM — pass it down and shut down gracefully.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx = runtime.ListenOSKillSignals(ctx)

	ids := slices.Map([]int{1, 2, 3}, func(i int) string {
		return "worker-" + string(rune('0'+i))
	})

	logger.
		WithFields(log.Fields{"workers": cfg.Workers, "ids": ids}).
		Info("service started")

	<-ctx.Done()
	logger.Info("service stopped")
}
```

## ⚒️ Local Development

### Prerequisites

- [mise](https://mise.jdx.dev)

### First Steps

```shell
git clone https://github.com/nightnoryu/go-kita
cd go-kita

# Build, lint and run unit tests
mise run
```

## 📜 License

Distributed under the MIT License. See [License](/LICENSE) for more information.

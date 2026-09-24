<p align="center">
	<img src="https://github.com/user-attachments/assets/07235b5a-1a5e-467d-a7f9-a8662d6d0e24" width="180" title="Go Kita Logo">
</p>

<h1 align="center">Go Kita</h1>

<p align="center">
    <a href="https://github.com/nightnoryu/go-kita/releases"><img src="https://img.shields.io/github/release/nightnoryu/go-kita.svg?cache-control=no-cache"></a>
    <a href="https://pkg.go.dev/github.com/nightnoryu/go-kita"><img src="https://pkg.go.dev/badge/github.com/nightnoryu/go-kita.svg?cache-control=no-cache"></a>
    <a href="https://github.com/nightnoryu/go-kita/blob/main/LICENSE"><img src="https://img.shields.io/github/license/nightnoryu/go-kita?cache-control=no-cache"></a>
	<a href="https://github.com/nightnoryu/go-kita/graphs/commit-activity" target="_blank"><img src="https://img.shields.io/github/commit-activity/m/nightnoryu/go-kita" alt="GitHub commit activity"></a>
    <a href="https://github.com/nightnoryu/go-kita/actions/workflows/ci.yml"><img src="https://github.com/nightnoryu/go-kita/actions/workflows/ci.yml/badge.svg?cache-control=no-cache"></a>
</p>

A lightweight SDK for building microservices without rewriting the same infrastructure boilerplate.

## 🎯 Why

I kept rewriting the same infrastructure code across my Go projects.
Go Kita is an attempt to extract those patterns into a small, reusable toolkit.

The project also serves as a playground for experimenting with microservice infrastructure.

> [!NOTE]
> Go Kita is primarily built for my own side projects and is not intended
> to be a general-purpose production framework.

## ✨ Features

- Structured JSON logging based on [zap](https://github.com/uber-go/zap)
- PostgreSQL connectivity with a custom migrator and a `database/sql` pool
  managed through [sqlx](https://github.com/jmoiron/sqlx), using the
  [pgx](https://github.com/jackc/pgx) driver
- Redis connectivity based on [go-redis](https://github.com/redis/go-redis)
- Environment variables parsing based on [caarlos0/env](https://github.com/caarlos0/env)
- Router-agnostic health and readiness handlers
- Maybe monad implementation with distinct Absent and None value semantics
- Context-aware retry scheduling with caller-defined retry policies
- ... and a couple more

Check the README in each package for more details. Also see
[`examples/service`](examples/service) for a complete service wiring example.

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
	"github.com/nightnoryu/go-kita/slices"
)

type config struct {
	LogLevel jsonlog.Level `env:"LOG_LEVEL" envDefault:"info"`
	Workers  int           `env:"WORKERS" envDefault:"4"`
}

func main() {
	// Reads MYAPP_LOG_LEVEL, MYAPP_WORKERS.
	cfg, err := env.ParseEnv[config]("myapp")
	if err != nil {
		panic(err)
	}

	logger, err := jsonlog.NewLogger(&jsonlog.Config{
		Level:   cfg.LogLevel,
		AppName: "myapp",
	})
	if err != nil {
		panic(err)
	}
	defer func() { _ = logger.Sync() }()

	ids := slices.Map([]int{1, 2, 3}, func(i int) string {
		return "worker-" + string(rune('0'+i))
	})

	logger.
		WithFields(log.Fields{"workers": cfg.Workers, "ids": ids}).
		Info("service started")
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

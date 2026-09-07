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

## 🛠 Local Development

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

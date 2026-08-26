<div align="center">
  <img src="https://github.com/user-attachments/assets/07235b5a-1a5e-467d-a7f9-a8662d6d0e24" alt="Logo" width="100" />
  <h1>Go Kita</h1>
</div>

[![Go Reference](https://pkg.go.dev/badge/github.com/nightnoryu/go-kita.svg)](https://pkg.go.dev/github.com/nightnoryu/go-kita)
[![Build Status](https://github.com/nightnoryu/go-kita/actions/workflows/validate.yml/badge.svg)](https://github.com/nightnoryu/go-kita/actions/workflows/validate.yml)

Go Kita is an SDK for building Go microservices.

## Goals

- Eliminate repetitive infrastructure plumbing
- Reuse recurring patterns in my projects

This is **not** a production-grade kit, I use this exclusively in my Go pet projects.

## Features

- Structured JSON logging based on [zap](https://github.com/uber-go/zap)
- Maybe monad implementation with distinct Absent and None value semantics
- PostgreSQL connectivity with custom migrator and connection pool based on [pgx](https://github.com/jackc/pgx)
- Redis connectivity based on [go-redis](https://github.com/redis/go-redis)
- Generic slices functions

Check README in each package for more details.

## License

Distributed under the MIT License. See [License](/LICENSE) for more information.

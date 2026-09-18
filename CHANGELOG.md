# Changelog

## v1.7.0

- `health` package added with concurrent, timeout-bound liveness and readiness
  checks plus composable HTTP handlers
- `redis.Client.Ping` and context-aware PostgreSQL ping support added for
  readiness checks

## v1.6.0

- `postgresql` connector startup now verifies connectivity with a context-aware
  timeout, safely cleans up failed handles, and supports pool configuration
- `postgresql` migrations now accept a context, use configurable advisory-lock
  IDs, validate and numerically order migration files, and preserve cleanup
  errors

## v1.5.0

- `jsonlog` now logs time in RFC3339 format with `time` field name instead of standard `ts`

## v1.4.0

- `runtime` package added

## v1.3.0

- `env` package added
- logger optimizations - added buffer flushing function and removed reflection usage on common types
- minor fixes in `postgresql` migrator

## v1.2.1

- `executor.ExecuteWithLock` now recovers gracefully in case of panic

## v1.2.0

- `slices.MapErr` added
- redis configuration divided in separate DNS and config
- added max connections and connection lifetime settings for redis client

## v1.1.0

- `redis` package added

## v1.0.0

Initial release

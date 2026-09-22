# Service example

This runnable program composes environment configuration, structured logging,
PostgreSQL migrations, Redis startup validation, health endpoints, and graceful
HTTP shutdown. It has no application routes beyond `/healthz` and `/readyz`.

Start PostgreSQL and Redis, then run:

```sh
EXAMPLE_SERVICE_POSTGRES_HOST=localhost \
EXAMPLE_SERVICE_POSTGRES_DATABASE=example \
EXAMPLE_SERVICE_POSTGRES_USER=postgres \
EXAMPLE_SERVICE_POSTGRES_PASSWORD=postgres \
EXAMPLE_SERVICE_REDIS_HOST=localhost \
go run ./examples/service
```

Optional variables are `EXAMPLE_SERVICE_LISTEN_ADDRESS` (default `:8080`),
`EXAMPLE_SERVICE_LOG_LEVEL` (default `info`),
`EXAMPLE_SERVICE_SHUTDOWN_TIMEOUT` (default `10s`),
`EXAMPLE_SERVICE_POSTGRES_PORT` (default `5432`),
`EXAMPLE_SERVICE_REDIS_PORT` (default `6379`), `EXAMPLE_SERVICE_REDIS_PASSWORD`,
and `EXAMPLE_SERVICE_REDIS_DB` (default `0`).

Send `SIGINT` or `SIGTERM` to stop accepting HTTP requests, wait for in-flight
requests, then close the Redis client and PostgreSQL connector. The program
joins cleanup errors with the operation error so shutdown failures remain
visible.

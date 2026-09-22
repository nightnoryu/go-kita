# redis

`redis` provides a small, context-aware wrapper around the Redis commands.

`NewClient` creates a lazy client. Use `OpenClient` during startup when Redis
must be reachable before the service begins accepting work. Both return a
client owned by the caller, which must call `Close` during shutdown.

`Config` has useful zero defaults. `DSN` must supply a host and port;
`DSN.Addr` does not validate either field. Clients are safe for concurrent
command use according to go-redis. Do not call commands after `Close`, and
coordinate shutdown so in-flight application work has stopped before closing
the client. `Close` returns any error reported by go-redis while releasing its
pool resources.

```go
cache, err := redis.OpenClient(ctx, redis.DSN{Host: "cache", Port: 6379}, redis.Config{
	DialTimeout: time.Second,
	TLSConfig:   &tls.Config{MinVersion: tls.VersionTLS12},
})
if err != nil {
	return err
}
defer cache.Close()
```

All commands accept a caller-supplied context. Context cancellation and
deadlines bound socket operations; a caller deadline sooner than the configured
read or write timeout takes precedence.

`Config` uses go-redis defaults for zero timeout values: five seconds for dial
and read, read timeout for write, and read timeout plus one second for pool
waits. A zero `MaxActiveConnections` leaves the pool unlimited; a zero
`ConnectionMaxLifetime` disables age-based connection expiry. `TLSConfig`
enables TLS when non-nil and is shallow-cloned when the client is created.
Configure certificate roots and server-name verification with the
standard-library `tls.Config`, and do not mutate data referenced by that config
after creating the client.

`Get` returns `ErrKeyNotFound` for a missing key. Empty `Delete` and `ZRem`
calls are successful no-ops.

`Client` is a convenience interface, not a required application boundary.
Applications should define a narrow local interface for the operations they
need, for example:

```go
type pinger interface {
	Ping(context.Context) error
}
```

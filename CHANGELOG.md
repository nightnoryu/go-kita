# Changelog

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

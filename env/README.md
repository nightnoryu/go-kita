# env

Environment-based configuration parsing built on [caarlos0/env](https://github.com/caarlos0/env).

`ParseEnv` populates a struct of type `T` from environment variables. Every variable is looked up with the prefix
`<APP_ID>_`, where `APP_ID` is the `appID` argument uppercased with hyphens converted to underscores. For example,
`my-app` uses the `MY_APP_` prefix. Field mapping, defaults, and validation are driven by the `env` struct tags of the
underlying library.

## Example

```go
package main

import (
	"github.com/nightnoryu/go-kita/env"
)

type config struct {
	Host  string `env:"DB_HOST,required"`
	Port  int    `env:"DB_PORT" envDefault:"5432"`
	Debug bool   `env:"DEBUG" envDefault:"false"`
}

func main() {
	// Reads MYAPP_DB_HOST, MYAPP_DB_PORT, MYAPP_DEBUG.
	cfg, err := env.ParseEnv[config]("myapp")
	if err != nil {
		panic(err)
	}

	_ = cfg
}

```

## Notes

- The prefix is always `strings.ToUpper(strings.ReplaceAll(appID, "-", "_")) + "_"`; passing an empty `appID`
  results in a bare `_` prefix.
- Parse failures (missing `required` fields, unparseable values) are returned wrapped with the message
  `failed to parse env`; the returned pointer is `nil` in that case.
- See the [caarlos0/env tag reference](https://github.com/caarlos0/env#tags) for `envDefault`, `required`,
  `envSeparator`, and other options.

## Nested service configuration

Keep environment-specific structs in the application, then map them to client configuration. This keeps the `env`
package independent of PostgreSQL and Redis clients.

```go
type config struct {
	LogLevel        jsonlog.Level `env:"LOG_LEVEL" envDefault:"info"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	Postgres        postgresConfig `envPrefix:"POSTGRES_"`
	Redis           redisConfig    `envPrefix:"REDIS_"`
}

type postgresConfig struct {
	Host           string        `env:"HOST,required"`
	Port           int           `env:"PORT" envDefault:"5432"`
	Database       string        `env:"DATABASE,required"`
	User           string        `env:"USER,required"`
	Password       string        `env:"PASSWORD,required"`
	ConnectTimeout time.Duration `env:"CONNECT_TIMEOUT" envDefault:"5s"`
}

type redisConfig struct {
	Host     string `env:"HOST,required"`
	Port     int    `env:"PORT" envDefault:"6379"`
	Password string `env:"PASSWORD"`
	DB       int    `env:"DB" envDefault:"0"`
}

func (c postgresConfig) dsn() postgresql.DSN {
	return postgresql.DSN{
		Host: c.Host, Port: c.Port, Database: c.Database, User: c.User, Password: c.Password,
	}
}

func (c postgresConfig) clientConfig() postgresql.Config {
	return postgresql.Config{ConnectTimeout: c.ConnectTimeout}
}

func (c redisConfig) dsn() redis.DSN {
	return redis.DSN{Host: c.Host, Port: c.Port, Password: c.Password, DB: c.DB}
}
```

For an application ID of `my-app`, these fields read values such as `MY_APP_POSTGRES_HOST` and
`MY_APP_REDIS_PORT`. The example requires imports for `time`, `jsonlog`, `postgresql`, and `redis`.

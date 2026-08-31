# env

Environment-based configuration parsing built on [caarlos0/env](https://github.com/caarlos0/env).

`ParseEnv` populates a struct of type `T` from environment variables. Every variable is looked up with the prefix
`<APP_ID>_`, where `APP_ID` is the `appID` argument uppercased. Field mapping, defaults, and validation are driven by
the `env` struct tags of the underlying library.

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

- The prefix is always `strings.ToUpper(appID) + "_"`; passing an empty `appID` results in a bare `_` prefix.
- Parse failures (missing `required` fields, unparseable values) are returned wrapped with the message
  `failed to parse env`; the returned pointer is `nil` in that case.
- See the [caarlos0/env tag reference](https://github.com/caarlos0/env#tags) for `envDefault`, `required`,
  `envSeparator`, and other options.

package redis

import (
	"fmt"
)

type DSN struct {
	Host     string
	Port     int
	Password string
	DB       int
}

func (c DSN) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

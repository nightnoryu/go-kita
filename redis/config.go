package redis

import "fmt"

type Config struct {
	Host     string
	Port     int
	Password string
	DB       int
}

func (c Config) addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

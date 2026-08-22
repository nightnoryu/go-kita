package postgresql

import (
	"fmt"
	"net/url"
)

type DSN struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
}

func (d DSN) String() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(d.User, d.Password),
		Host:   fmt.Sprintf("%s:%d", d.Host, d.Port),
		Path:   "/" + d.Database,
	}
	return u.String()
}

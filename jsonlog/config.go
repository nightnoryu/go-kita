// Package jsonlog provides structured JSON logging backed by zap.
package jsonlog

type Config struct {
	Level   Level
	AppName string
}

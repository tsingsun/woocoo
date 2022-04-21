package log

import (
	"go.uber.org/zap"
)

const (
	WebComponentName = "web"
)

// ComponentLogger is sample and base using for component
type ComponentLogger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	DPanic(msg string, fields ...zap.Field)
	Panic(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
}

type component struct {
	name string
}

func (c *component) Debug(msg string, fields ...zap.Field) {
	global.Debug(msg, append(fields, zap.String("component", c.name))...)
}

func (c *component) Info(msg string, fields ...zap.Field) {
	global.Info(msg, append(fields, zap.String("component", c.name))...)
}

func (c *component) Warn(msg string, fields ...zap.Field) {
	global.Warn(msg, append(fields, zap.String("component", c.name))...)
}

func (c *component) Error(msg string, fields ...zap.Field) {
	global.Error(msg, append(fields, zap.String("component", c.name))...)
}

func (c *component) DPanic(msg string, fields ...zap.Field) {
	global.DPanic(msg, append(fields, zap.String("component", c.name))...)
}

func (c *component) Panic(msg string, fields ...zap.Field) {
	global.Panic(msg, append(fields, zap.String("component", c.name))...)
}

func (c *component) Fatal(msg string, fields ...zap.Field) {
	global.Fatal(msg, append(fields, zap.String("component", c.name))...)
}

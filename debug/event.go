package debug

import "time"

type LogLevel string

const (
	INFO    LogLevel = "INFO"
	WARNING LogLevel = "WARNING"
	ERROR   LogLevel = "ERROR"
	DEBUG   LogLevel = "DEBUG"
)

type DebugEvent struct {
	ID        int
	Timestamp time.Time
	Level     LogLevel
	Type      string
	Message   string
}

package logger

type Level string

const (
	DebugLevel Level = "DEBUG"
	InfoLevel  Level = "INFO"
	WarnLevel  Level = "WARN"
	ErrorLevel Level = "ERROR"
)

type Log struct {
	SessionID string `json:"sessionId"`
	Level     Level  `json:"level"`
	Time      int64  `json:"time"`
	Message   string `json:"message"`
	Args      []any  `json:"args,omitempty"`
}

type Logger interface {
	Log(level Level, msg string, args ...any)
	Rotate() error
	// Stops the logger, including the workers and the closes file.
	Stop()
}

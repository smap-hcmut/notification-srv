package log

const (
	ModeProduction  = "production"
	ModeDevelopment = "development"
	EncodingConsole = "console"
	EncodingJSON    = "json"

	// TraceIDKey is the key for trace id in context
	TraceIDKey = "trace_id"
)

const (
	LevelDebug  = "debug"
	LevelInfo   = "info"
	LevelWarn   = "warn"
	LevelError  = "error"
	LevelFatal  = "fatal"
	LevelPanic  = "panic"
	LevelDPanic = "dpanic"
)

package rv

type LogLevel int

const (
	LogLevelSilence LogLevel = iota
	LogLevelInfo
	LogLevelDebug
)

type Logger interface {
	Printf(lvl LogLevel, format string, args ...any)
}

type LogFunc func(lvl LogLevel, format string, args ...any)

func (f LogFunc) Printf(lvl LogLevel, format string, args ...any) {
	f(lvl, format, args...)
}

func devNull(_ LogLevel, _ string, _ ...any) {}

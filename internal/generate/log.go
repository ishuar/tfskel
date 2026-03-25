package generate

// Logger defines the logging interface used by the generator.
// Defined here per Go convention: interfaces belong in the consumer package.
type Logger interface {
	Debug(message string)
	Debugf(format string, args ...any)
	Info(message string)
	Infof(format string, args ...any)
	Warn(message string)
	Warnf(format string, args ...any)
	Success(message string)
	Successf(format string, args ...any)
	Error(message string)
	Errorf(format string, args ...any)
}

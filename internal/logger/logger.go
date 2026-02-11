package logger

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogLevel represents the level of logging
type LogLevel int

const (
	// DebugLevel logs verbose details with timestamps
	DebugLevel LogLevel = iota
	// InfoLevel logs general informational messages
	InfoLevel
	// WarnLevel logs non-critical issues
	WarnLevel
	// SuccessLevel logs positive confirmations
	SuccessLevel
	// ErrorLevel logs errors that may require user action
	ErrorLevel
)

// Logger provides structured logging with color output
type Logger struct {
	level  LogLevel
	out    io.Writer
	errOut io.Writer
}

// Color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
)

// New creates a new logger instance
// verbose flag enables DEBUG level and timestamps on all logs
func New(verbose bool) *Logger {
	level := InfoLevel
	if verbose {
		level = DebugLevel
	}

	// Check TFSKEL_LOG_LEVEL environment variable
	if envLevel := os.Getenv("TFSKEL_LOG_LEVEL"); envLevel != "" {
		if parsedLevel, ok := parseLogLevel(envLevel); ok {
			level = parsedLevel
		}
	}

	return &Logger{
		level:  level,
		out:    os.Stdout,
		errOut: os.Stderr,
	}
}

// NewWithWriters creates a logger with custom writers (useful for testing)
func NewWithWriters(verbose bool, out, errOut io.Writer) *Logger {
	level := InfoLevel
	if verbose {
		level = DebugLevel
	}

	// Check TFSKEL_LOG_LEVEL environment variable
	if envLevel := os.Getenv("TFSKEL_LOG_LEVEL"); envLevel != "" {
		if parsedLevel, ok := parseLogLevel(envLevel); ok {
			level = parsedLevel
		}
	}

	return &Logger{
		level:  level,
		out:    out,
		errOut: errOut,
	}
}

// SetOutput sets the output writer for info-level logs
func (l *Logger) SetOutput(w io.Writer) {
	l.out = w
}

// parseLogLevel converts string log level to LogLevel
func parseLogLevel(level string) (LogLevel, bool) {
	switch strings.ToLower(level) {
	case "debug":
		return DebugLevel, true
	case "info":
		return InfoLevel, true
	case "warn", "warning":
		return WarnLevel, true
	case "success":
		return SuccessLevel, true
	case "error":
		return ErrorLevel, true
	default:
		return InfoLevel, false
	}
}

// log is the internal logging function
func (l *Logger) log(color, levelStr, message string, out io.Writer, msgLevel LogLevel) {
	// Only log if the message level is at or above the configured level
	if msgLevel < l.level {
		return
	}

	var prefix string

	// Add timestamp and detailed info for DEBUG level
	if l.level == DebugLevel {
		timestamp := time.Now().Format("2006-01-02T15:04:05")
		prefix = fmt.Sprintf("%s%s ", color, timestamp)

		// Add caller info for DEBUG messages
		if msgLevel == DebugLevel {
			if _, file, line, ok := runtime.Caller(2); ok {
				// Extract just the filename without full path
				parts := strings.Split(file, "/")
				filename := parts[len(parts)-1]
				message = fmt.Sprintf("%s:%d %s", filename, line, message)
			}
		}
	} else {
		prefix = color
	}

	_, _ = fmt.Fprintf(out, "%s[%-7s] %s%s\n", prefix, levelStr, message, colorReset) //nolint:errcheck // Writing to stderr/stdout in logger is best-effort, ignoring errors is acceptable
}

// Debug logs a debug message only if debug level is enabled
// Shows timestamps, file locations, and detailed internals
func (l *Logger) Debug(message string) {
	l.log(colorBlue, "DEBUG", message, l.out, DebugLevel)
}

// Debugf logs a formatted debug message
func (l *Logger) Debugf(format string, args ...any) {
	l.Debug(fmt.Sprintf(format, args...))
}

// Info logs an informational message
// No timestamps unless in debug mode
func (l *Logger) Info(message string) {
	l.log(colorCyan, "INFO", message, l.out, InfoLevel)
}

// Infof logs a formatted info message
func (l *Logger) Infof(format string, args ...any) {
	l.Info(fmt.Sprintf(format, args...))
}

// Warn logs a warning message for non-critical issues
// No timestamps unless in debug mode
func (l *Logger) Warn(message string) {
	l.log(colorYellow, "WARN", message, l.out, WarnLevel)
}

// Warnf logs a formatted warning message
func (l *Logger) Warnf(format string, args ...any) {
	l.Warn(fmt.Sprintf(format, args...))
}

// Success logs a success message in green
// No timestamps unless in debug mode
func (l *Logger) Success(message string) {
	l.log(colorGreen, "SUCCESS", message, l.out, SuccessLevel)
}

// Successf logs a formatted success message
func (l *Logger) Successf(format string, args ...any) {
	l.Success(fmt.Sprintf(format, args...))
}

// Error logs an error message in red to stderr
// No timestamps unless in debug mode
func (l *Logger) Error(message string) {
	l.log(colorRed, "ERROR", message, l.errOut, ErrorLevel)
}

// Errorf logs a formatted error message
func (l *Logger) Errorf(format string, args ...any) {
	l.Error(fmt.Sprintf(format, args...))
}

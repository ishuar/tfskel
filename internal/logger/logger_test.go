package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	t.Run("Success message", func(t *testing.T) {
		var out bytes.Buffer
		log := NewWithWriters(false, &out, &out)
		log.Success("test message")

		output := out.String()
		assert.Contains(t, output, "SUCCESS")
		assert.Contains(t, output, "test message")
		assert.NotContains(t, output, "T", "Should not contain timestamp in non-verbose mode")
	})

	t.Run("Info message", func(t *testing.T) {
		var out bytes.Buffer
		log := NewWithWriters(false, &out, &out)
		log.Info("test info")

		output := out.String()
		assert.Contains(t, output, "INFO")
		assert.Contains(t, output, "test info")
		assert.NotContains(t, output, "202", "Should not contain timestamp in non-verbose mode")
	})

	t.Run("Warn message", func(t *testing.T) {
		var out bytes.Buffer
		log := NewWithWriters(false, &out, &out)
		log.Warn("test warning")

		output := out.String()
		assert.Contains(t, output, "WARN")
		assert.Contains(t, output, "test warning")
		assert.NotContains(t, output, "202", "Should not contain timestamp in non-verbose mode")
	})

	t.Run("Error message", func(t *testing.T) {
		var errOut bytes.Buffer
		log := NewWithWriters(false, &bytes.Buffer{}, &errOut)
		log.Error("test error")

		output := errOut.String()
		assert.Contains(t, output, "ERROR")
		assert.Contains(t, output, "test error")
		assert.NotContains(t, output, "202", "Should not contain timestamp in non-verbose mode")
	})

	t.Run("Debug message with verbose off", func(t *testing.T) {
		var out bytes.Buffer
		log := NewWithWriters(false, &out, &out)
		log.Debug("debug message")

		output := out.String()
		assert.Empty(t, output, "Debug message should not be logged when verbose is off")
	})

	t.Run("Debug message with verbose on", func(t *testing.T) {
		var out bytes.Buffer
		log := NewWithWriters(true, &out, &out)
		log.Debug("debug message")

		output := out.String()
		assert.Contains(t, output, "DEBUG")
		assert.Contains(t, output, "debug message")
		assert.Contains(t, output, "202", "Should contain timestamp in verbose mode")
		assert.Contains(t, output, "logger_test.go", "Should contain filename in debug mode")
	})

	t.Run("Formatted messages", func(t *testing.T) {
		var out bytes.Buffer
		log := NewWithWriters(false, &out, &out)
		log.Infof("formatted %s with %d", "message", 42)

		output := out.String()
		assert.Contains(t, output, "formatted message with 42")
	})

	t.Run("Formatted warning", func(t *testing.T) {
		var out bytes.Buffer
		log := NewWithWriters(false, &out, &out)
		log.Warnf("warning %s", "test")

		output := out.String()
		assert.Contains(t, output, "WARN")
		assert.Contains(t, output, "warning test")
	})

	t.Run("Formatted debug", func(t *testing.T) {
		var out bytes.Buffer
		log := NewWithWriters(true, &out, &out)
		log.Debugf("debug %d", 123)

		output := out.String()
		assert.Contains(t, output, "DEBUG")
		assert.Contains(t, output, "debug 123")
	})

	t.Run("Formatted success", func(t *testing.T) {
		var out bytes.Buffer
		log := NewWithWriters(false, &out, &out)
		log.Successf("success %s", "operation")

		output := out.String()
		assert.Contains(t, output, "SUCCESS")
		assert.Contains(t, output, "success operation")
	})

	t.Run("Formatted error", func(t *testing.T) {
		var errOut bytes.Buffer
		log := NewWithWriters(false, &bytes.Buffer{}, &errOut)
		log.Errorf("error %s", "occurred")

		output := errOut.String()
		assert.Contains(t, output, "ERROR")
		assert.Contains(t, output, "error occurred")
	})
}

func TestNew(t *testing.T) {
	t.Run("creates logger with default settings", func(t *testing.T) {
		log := New(false)
		assert.NotNil(t, log)
		assert.Equal(t, InfoLevel, log.level)
		assert.Equal(t, os.Stdout, log.out)
		assert.Equal(t, os.Stderr, log.errOut)
	})

	t.Run("creates logger with verbose enabled", func(t *testing.T) {
		log := New(true)
		assert.NotNil(t, log)
		assert.Equal(t, DebugLevel, log.level)
	})

	t.Run("respects TFSKEL_LOG_LEVEL environment variable", func(t *testing.T) {
		err := os.Setenv("TFSKEL_LOG_LEVEL", "warn")
		assert.NoError(t, err)
		defer func() {
			err := os.Unsetenv("TFSKEL_LOG_LEVEL")
			assert.NoError(t, err)
		}()

		log := New(false)
		assert.Equal(t, WarnLevel, log.level)
	})
}

func TestSetOutput(t *testing.T) {
	t.Run("changes output writer", func(t *testing.T) {
		var out1, out2 bytes.Buffer
		log := NewWithWriters(false, &out1, &out1)
		log.Info("first")

		log.SetOutput(&out2)
		log.Info("second")

		output1 := out1.String()
		output2 := out2.String()

		assert.Contains(t, output1, "first")
		assert.NotContains(t, output1, "second")
		assert.Contains(t, output2, "second")
		assert.NotContains(t, output2, "first")
	})
}

func TestLoggerTimestamp(t *testing.T) {
	t.Run("Timestamp in verbose mode", func(t *testing.T) {
		var out bytes.Buffer
		log := NewWithWriters(true, &out, &out)
		log.Info("test")

		output := out.String()
		// Check that timestamp format is present (YYYY-MM-DD)
		assert.Contains(t, output, "202", "Should contain timestamp in verbose mode")
		assert.Contains(t, output, "T", "Should contain ISO8601 timestamp format")
	})

	t.Run("No timestamp in normal mode", func(t *testing.T) {
		var out bytes.Buffer
		log := NewWithWriters(false, &out, &out)
		log.Info("test")

		output := out.String()
		// Should not have timestamp pattern
		lines := strings.Split(strings.TrimSpace(output), "\n")
		assert.Equal(t, 1, len(lines))
		assert.NotContains(t, output, "T", "Should not contain timestamp in normal mode")
	})
}

func TestLoggerLevels(t *testing.T) {
	t.Run("Debug level shows all messages", func(t *testing.T) {
		var out, errOut bytes.Buffer
		log := NewWithWriters(true, &out, &errOut)

		log.Debug("debug msg")
		log.Info("info msg")
		log.Warn("warn msg")
		log.Success("success msg")
		log.Error("error msg")

		stdoutOutput := out.String()
		stderrOutput := errOut.String()

		assert.Contains(t, stdoutOutput, "DEBUG")
		assert.Contains(t, stdoutOutput, "INFO")
		assert.Contains(t, stdoutOutput, "WARN")
		assert.Contains(t, stdoutOutput, "SUCCESS")
		assert.Contains(t, stderrOutput, "ERROR")
	})

	t.Run("Info level hides debug", func(t *testing.T) {
		var out, errOut bytes.Buffer
		log := NewWithWriters(false, &out, &errOut)

		log.Debug("debug msg")
		log.Info("info msg")
		log.Warn("warn msg")

		output := out.String()
		assert.NotContains(t, output, "DEBUG")
		assert.Contains(t, output, "INFO")
		assert.Contains(t, output, "WARN")
	})
}

func TestLoggerEnvironmentVariable(t *testing.T) {
	t.Run("TFSKEL_LOG_LEVEL=debug enables debug", func(t *testing.T) {
		err := os.Setenv("TFSKEL_LOG_LEVEL", "debug")
		assert.NoError(t, err)
		defer func() {
			err := os.Unsetenv("TFSKEL_LOG_LEVEL")
			assert.NoError(t, err)
		}()

		var out bytes.Buffer
		log := NewWithWriters(false, &out, &out)
		log.Debug("debug message")

		output := out.String()
		assert.Contains(t, output, "DEBUG")
		assert.Contains(t, output, "debug message")
	})

	t.Run("TFSKEL_LOG_LEVEL=info", func(t *testing.T) {
		err := os.Setenv("TFSKEL_LOG_LEVEL", "info")
		assert.NoError(t, err)
		defer func() {
			err := os.Unsetenv("TFSKEL_LOG_LEVEL")
			assert.NoError(t, err)
		}()

		var out bytes.Buffer
		log := NewWithWriters(true, &out, &out) // verbose=true but env overrides

		log.Debug("debug message")
		log.Info("info message")

		output := out.String()
		assert.NotContains(t, output, "DEBUG", "Debug should be hidden at info level")
		assert.Contains(t, output, "INFO")
	})

	t.Run("Invalid TFSKEL_LOG_LEVEL falls back to info", func(t *testing.T) {
		err := os.Setenv("TFSKEL_LOG_LEVEL", "invalid")
		assert.NoError(t, err)
		defer func() {
			err := os.Unsetenv("TFSKEL_LOG_LEVEL")
			assert.NoError(t, err)
		}()

		var out bytes.Buffer
		log := NewWithWriters(false, &out, &out)
		log.Debug("debug message")
		log.Info("info message")

		output := out.String()
		assert.NotContains(t, output, "DEBUG")
		assert.Contains(t, output, "INFO")
	})
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
		valid    bool
	}{
		{"debug", DebugLevel, true},
		{"DEBUG", DebugLevel, true},
		{"info", InfoLevel, true},
		{"INFO", InfoLevel, true},
		{"warn", WarnLevel, true},
		{"warning", WarnLevel, true},
		{"WARN", WarnLevel, true},
		{"success", SuccessLevel, true},
		{"error", ErrorLevel, true},
		{"invalid", InfoLevel, false},
		{"", InfoLevel, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level, valid := parseLogLevel(tt.input)
			assert.Equal(t, tt.expected, level)
			assert.Equal(t, tt.valid, valid)
		})
	}
}

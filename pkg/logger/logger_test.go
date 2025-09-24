package logger

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestStandardLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStandardLogger(log.New(&buf, "", 0), Debug, "[test]")

	t.Run("Info", func(t *testing.T) {
		buf.Reset()
		logger.Info("info message", "key1", "value1", "key2", 123)
		output := buf.String()
		if !strings.Contains(output, "[test] [INFO] info message") {
			t.Errorf("Expected log message not found in: %s", output)
		}
		if !strings.Contains(output, "key1=value1") || !strings.Contains(output, "key2=123") {
			t.Errorf("Expected structured fields not found in: %s", output)
		}
	})

	t.Run("Debug", func(t *testing.T) {
		buf.Reset()
		logger.Debug("debug message")
		if !strings.Contains(buf.String(), "[DEBUG] debug message") {
			t.Errorf("Expected log message not found in: %s", buf.String())
		}
	})
}

func TestStandardLogger_Levels(t *testing.T) {
	var buf bytes.Buffer
	warnLogger := NewStandardLogger(log.New(&buf, "", 0), Warn, "[test]")

	// This should not be logged
	warnLogger.Info("info message")
	if buf.Len() > 0 {
		t.Errorf("Info should not be logged at Warn level, but got: %s", buf.String())
	}

	// This should be logged
	warnLogger.Warn("warn message")
	if !strings.Contains(buf.String(), "[WARN] warn message") {
		t.Errorf("Warn should be logged at Warn level, but got: %s", buf.String())
	}
}

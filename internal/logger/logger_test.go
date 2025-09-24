package logger

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	// Test logger creation
	logger := NewLogger("test-service")
	
	require.NotNil(t, logger)
	require.NotNil(t, logger.logger)
	assert.Equal(t, "test-service", logger.logger.GetString("service"))
}

func TestLogger_Info(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Create logger
	logger := NewLogger("test-service")

	// Test info logging
	event := logger.Info()
	require.NotNil(t, event)
	
	// Send the log
	event.Msg("Test info message")
	
	// Check the output
	output := buf.String()
	assert.Contains(t, output, "Test info message")
	assert.Contains(t, output, "test-service")
	assert.Contains(t, output, "info")
}

func TestLogger_Error(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Create logger
	logger := NewLogger("test-service")

	// Test error logging
	event := logger.Error()
	require.NotNil(t, event)
	
	// Send the log with an error
	event.Msg("Test error message")
	
	// Check the output
	output := buf.String()
	assert.Contains(t, output, "Test error message")
	assert.Contains(t, output, "test-service")
	assert.Contains(t, output, "error")
}

func TestLogger_Debug(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Create logger
	logger := NewLogger("test-service")

	// Test debug logging
	event := logger.Debug()
	require.NotNil(t, event)
	
	// Send the log
	event.Msg("Test debug message")
	
	// Check the output
	output := buf.String()
	assert.Contains(t, output, "Test debug message")
	assert.Contains(t, output, "test-service")
	assert.Contains(t, output, "debug")
}

func TestLogger_Warn(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Create logger
	logger := NewLogger("test-service")

	// Test warn logging
	event := logger.Warn()
	require.NotNil(t, event)
	
	// Send the log
	event.Msg("Test warn message")
	
	// Check the output
	output := buf.String()
	assert.Contains(t, output, "Test warn message")
	assert.Contains(t, output, "test-service")
	assert.Contains(t, output, "warn")
}

func TestLogger_Fatal(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Create logger
	logger := NewLogger("test-service")

	// Test fatal logging
	event := logger.Fatal()
	require.NotNil(t, event)
	
	// Send the log
	event.Msg("Test fatal message")
	
	// Check the output
	output := buf.String()
	assert.Contains(t, output, "Test fatal message")
	assert.Contains(t, output, "test-service")
	assert.Contains(t, output, "fatal")
}

func TestLogger_Panic(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Create logger
	logger := NewLogger("test-service")

	// Test panic logging
	event := logger.Panic()
	require.NotNil(t, event)
	
	// Send the log
	event.Msg("Test panic message")
	
	// Check the output
	output := buf.String()
	assert.Contains(t, output, "Test panic message")
	assert.Contains(t, output, "test-service")
	assert.Contains(t, output, "panic")
}

func TestLogger_InfoRequest(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Create logger
	logger := NewLogger("test-service")

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	req.Header.Set("User-Agent", "test-agent")

	// Test request logging
	logger.InfoRequest(req, "Test request message")
	
	// Check the output
	output := buf.String()
	assert.Contains(t, output, "Test request message")
	assert.Contains(t, output, "test-service")
	assert.Contains(t, output, "GET")
	assert.Contains(t, output, "/test")
	assert.Contains(t, output, "192.168.1.100")
	assert.Contains(t, output, "test-agent")
}

func TestLogger_InfoRequest_WithFields(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Create logger
	logger := NewLogger("test-service")

	// Create a test request
	req := httptest.NewRequest("POST", "/api/cis", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	// Test request logging with fields
	fields := map[string]interface{}{
		"request_id": "test-123",
		"user_id":    "user-456",
		"duration":   123 * time.Millisecond,
	}
	logger.InfoRequest(req, "Test request with fields", fields)
	
	// Check the output
	output := buf.String()
	assert.Contains(t, output, "Test request with fields")
	assert.Contains(t, output, "test-service")
	assert.Contains(t, output, "POST")
	assert.Contains(t, output, "/api/cis")
	assert.Contains(t, output, "test-123")
	assert.Contains(t, output, "user-456")
	assert.Contains(t, output, "123ms")
}

func TestLogger_ErrorRequest(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Create logger
	logger := NewLogger("test-service")

	// Create a test request
	req := httptest.NewRequest("DELETE", "/api/cis/123", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	// Create a test error
	testErr := assert.AnError

	// Test error request logging
	logger.ErrorRequest(req, testErr, "Test error request")
	
	// Check the output
	output := buf.String()
	assert.Contains(t, output, "Test error request")
	assert.Contains(t, output, "test-service")
	assert.Contains(t, output, "DELETE")
	assert.Contains(t, output, "/api/cis/123")
	assert.Contains(t, output, testErr.Error())
}

func TestLogger_ErrorRequest_WithFields(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Create logger
	logger := NewLogger("test-service")

	// Create a test request
	req := httptest.NewRequest("PUT", "/api/cis/123", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	// Create a test error
	testErr := assert.AnError

	// Test error request logging with fields
	fields := map[string]interface{}{
		"request_id": "test-789",
		"error_code": "CI_NOT_FOUND",
		"retry_count": 3,
	}
	logger.ErrorRequest(req, testErr, "Test error request with fields", fields)
	
	// Check the output
	output := buf.String()
	assert.Contains(t, output, "Test error request with fields")
	assert.Contains(t, output, "test-service")
	assert.Contains(t, output, "PUT")
	assert.Contains(t, output, "/api/cis/123")
	assert.Contains(t, output, testErr.Error())
	assert.Contains(t, output, "test-789")
	assert.Contains(t, output, "CI_NOT_FOUND")
	assert.Contains(t, output, "3")
}

func TestLogger_InfoOperation(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Create logger
	logger := NewLogger("test-service")

	// Test operation logging
	duration := 456 * time.Millisecond
	logger.InfoOperation("database_query", duration, "Test operation message")
	
	// Check the output
	output := buf.String()
	assert.Contains(t, output, "Test operation message")
	assert.Contains(t, output, "test-service")
	assert.Contains(t, output, "database_query")
	assert.Contains(t, output, "456ms")
}

func TestLogger_InfoOperation_WithFields(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Create logger
	logger := NewLogger("test-service")

	// Test operation logging with fields
	duration := 789 * time.Millisecond
	fields := map[string]interface{}{
		"query":      "SELECT * FROM configuration_items",
		"rows":       100,
		"cache_hit":  true,
	}
	logger.InfoOperation("database_query", duration, "Test operation with fields", fields)
	
	// Check the output
	output := buf.String()
	assert.Contains(t, output, "Test operation with fields")
	assert.Contains(t, output, "test-service")
	assert.Contains(t, output, "database_query")
	assert.Contains(t, output, "789ms")
	assert.Contains(t, output, "SELECT * FROM configuration_items")
	assert.Contains(t, output, "100")
	assert.Contains(t, output, "true")
}

func TestLogger_ErrorOperation(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Create logger
	logger := NewLogger("test-service")

	// Create a test error
	testErr := assert.AnError

	// Test error operation logging
	duration := 123 * time.Millisecond
	logger.ErrorOperation("api_call", duration, testErr, "Test error operation")
	
	// Check the output
	output := buf.String()
	assert.Contains(t, output, "Test error operation")
	assert.Contains(t, output, "test-service")
	assert.Contains(t, output, "api_call")
	assert.Contains(t, output, "123ms")
	assert.Contains(t, output, testErr.Error())
}

func TestLogger_ErrorOperation_WithFields(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Create logger
	logger := NewLogger("test-service")

	// Create a test error
	testErr := assert.AnError

	// Test error operation logging with fields
	duration := 321 * time.Millisecond
	fields := map[string]interface{}{
		"endpoint":   "/api/cis",
		"status":     500,
		"retries":    2,
	}
	logger.ErrorOperation("api_call", duration, testErr, "Test error operation with fields", fields)
	
	// Check the output
	output := buf.String()
	assert.Contains(t, output, "Test error operation with fields")
	assert.Contains(t, output, "test-service")
	assert.Contains(t, output, "api_call")
	assert.Contains(t, output, "321ms")
	assert.Contains(t, output, testErr.Error())
	assert.Contains(t, output, "/api/cis")
	assert.Contains(t, output, "500")
	assert.Contains(t, output, "2")
}

func TestSetLogLevel(t *testing.T) {
	// Test setting log level
	err := SetLogLevel("debug")
	require.NoError(t, err)
	assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())

	err = SetLogLevel("info")
	require.NoError(t, err)
	assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())

	err = SetLogLevel("warn")
	require.NoError(t, err)
	assert.Equal(t, zerolog.WarnLevel, zerolog.GlobalLevel())

	err = SetLogLevel("error")
	require.NoError(t, err)
	assert.Equal(t, zerolog.ErrorLevel, zerolog.GlobalLevel())

	// Test invalid log level
	err = SetLogLevel("invalid")
	require.Error(t, err)
}

func TestSetLogFormat(t *testing.T) {
	// Test setting log format (this is mostly to ensure it doesn't panic)
	SetLogFormat("json")
	SetLogFormat("console")
	// The actual format testing would require more complex setup
}

func TestLogger_MultipleFields(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Create logger
	logger := NewLogger("test-service")

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	// Test request logging with multiple field maps
	fields1 := map[string]interface{}{
		"request_id": "test-123",
		"user_id":    "user-456",
	}
	fields2 := map[string]interface{}{
		"duration":   123 * time.Millisecond,
		"status":     200,
	}
	logger.InfoRequest(req, "Test multiple fields", fields1, fields2)
	
	// Check the output
	output := buf.String()
	assert.Contains(t, output, "Test multiple fields")
	assert.Contains(t, output, "test-123")
	assert.Contains(t, output, "user-456")
	assert.Contains(t, output, "123ms")
	assert.Contains(t, output, "200")
}

func TestLogger_EmptyFields(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Create logger
	logger := NewLogger("test-service")

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	// Test request logging with empty fields
	var emptyFields map[string]interface{}
	logger.InfoRequest(req, "Test empty fields", emptyFields)
	
	// Check the output
	output := buf.String()
	assert.Contains(t, output, "Test empty fields")
	assert.Contains(t, output, "test-service")
}

func TestLogger_NilFields(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer
	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Create logger
	logger := NewLogger("test-service")

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"

	// Test request logging with nil fields (this should not panic)
	logger.InfoRequest(req, "Test nil fields", nil)
	
	// Check the output
	output := buf.String()
	assert.Contains(t, output, "Test nil fields")
	assert.Contains(t, output, "test-service")
}

func TestLogger_GlobalLogger(t *testing.T) {
	// Test that the global logger is set correctly
	originalLogger := log.Logger
	
	// Create a buffer to capture log output
	var buf bytes.Buffer
	testLogger := zerolog.New(&buf).With().Timestamp().Logger()
	
	// Create new logger (this should set the global logger)
	NewLogger("test-service")
	
	// The global logger should be different from the original
	assert.NotEqual(t, originalLogger, log.Logger)
	
	// Restore original logger
	log.Logger = originalLogger
}

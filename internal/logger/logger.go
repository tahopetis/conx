package logger

import (
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Logger struct {
	logger *zerolog.Logger
}

func NewLogger(service string) *Logger {
	// Create a new logger
	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", service).
		Logger()

	// Set global logger
	log.Logger = logger

	return &Logger{logger: &logger}
}

func (l *Logger) Info() *zerolog.Event {
	return l.logger.Info()
}

func (l *Logger) Error() *zerolog.Event {
	return l.logger.Error()
}

func (l *Logger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

func (l *Logger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

func (l *Logger) Fatal() *zerolog.Event {
	return l.logger.Fatal()
}

func (l *Logger) Panic() *zerolog.Event {
	return l.logger.Panic()
}

func (l *Logger) InfoRequest(r *http.Request, message string, fields ...map[string]interface{}) {
	event := l.logger.Info().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("ip", r.RemoteAddr).
		Str("user_agent", r.UserAgent())

	for _, field := range fields {
		for k, v := range field {
			event = event.Interface(k, v)
		}
	}

	event.Msg(message)
}

func (l *Logger) ErrorRequest(r *http.Request, err error, message string, fields ...map[string]interface{}) {
	event := l.logger.Error().
		Err(err).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("ip", r.RemoteAddr)

	for _, field := range fields {
		for k, v := range field {
			event = event.Interface(k, v)
		}
	}

	event.Msg(message)
}

func (l *Logger) InfoOperation(operation string, duration time.Duration, message string, fields ...map[string]interface{}) {
	event := l.logger.Info().
		Str("operation", operation).
		Dur("duration", duration)

	for _, field := range fields {
		for k, v := range field {
			event = event.Interface(k, v)
		}
	}

	event.Msg(message)
}

func (l *Logger) ErrorOperation(operation string, duration time.Duration, err error, message string, fields ...map[string]interface{}) {
	event := l.logger.Error().
		Str("operation", operation).
		Dur("duration", duration).
		Err(err)

	for _, field := range fields {
		for k, v := range field {
			event = event.Interface(k, v)
		}
	}

	event.Msg(message)
}

// SetLogLevel sets the global log level
func SetLogLevel(level string) error {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(lvl)
	return nil
}

// SetLogFormat sets the log format
func SetLogFormat(format string) {
	if format == "json" {
		log.Logger = log.Logger.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}
}

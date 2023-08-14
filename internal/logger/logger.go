// Package logger provides logging functionality.

package logger

import (
	"os"
	"time"
	"upload-service-auto/internal/config"

	"github.com/rs/zerolog"
)

// NewLog initializes a logger.
func NewLog(cfg *config.Config) *zerolog.Logger {
	var level zerolog.Level
	switch cfg.Logger.Level {
	case 0:
		level = zerolog.DebugLevel
	case 1:
		level = zerolog.InfoLevel
	case 2:
		level = zerolog.WarnLevel
	case 3:
		level = zerolog.ErrorLevel
	default:
		level = zerolog.DebugLevel
	}

	zerolog.TimeFieldFormat = time.RFC3339
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}
	Logger := zerolog.New(consoleWriter).With().Timestamp().Logger().Level(level)
	return &Logger
}

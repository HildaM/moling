package comm

import (
	"context"
	"os"
	"path/filepath"

	"github.com/gojue/moling/pkg/config"
	"github.com/rs/zerolog"
)

// MoLingServerType is the type of the server
type MoLingServerType string

// contextKey is a type for context keys
type contextKey string

// MoLingConfigKey is a context key for storing the version of MoLing
const (
	MoLingConfigKey contextKey = "moling_config"
	MoLingLoggerKey contextKey = "moling_logger"
)

// InitTestEnv initializes the test environment by creating a temporary log file and setting up the logger.
func InitTestEnv() (zerolog.Logger, context.Context, error) {
	logFile := filepath.Join(os.TempDir(), "moling.log")
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	var logger zerolog.Logger
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return zerolog.Logger{}, nil, err
	}
	logger = zerolog.New(f).With().Timestamp().Logger()
	mlConfig := &config.MoLingConfig{
		ConfigFile: filepath.Join("config", "test_config.json"),
		BasePath:   os.TempDir(),
	}
	ctx := context.WithValue(context.Background(), MoLingConfigKey, mlConfig)
	ctx = context.WithValue(ctx, MoLingLoggerKey, logger)
	return logger, ctx, nil
}

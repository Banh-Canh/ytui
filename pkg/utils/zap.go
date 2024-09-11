package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func InitializeLogger(logLevel zapcore.Level, logFilePath string) {
	// Create all necessary directories for the log file
	dir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		panic(fmt.Sprintf("Failed to create log directory: %v", err))
	}

	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(logLevel), // Set the log level
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:      "json",
		EncoderConfig: zap.NewProductionEncoderConfig(),

		OutputPaths:      []string{logFilePath},
		ErrorOutputPaths: []string{logFilePath},
	}

	var err error
	Logger, err = config.Build()
	if err != nil {
		panic(err)
	}
	defer Logger.Sync() //nolint:all
}

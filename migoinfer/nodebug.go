// +build !debug

package migoinfer

import (
	"log"

	"github.com/fatih/color"
	"github.com/nickng/gospal/migoinfer/internal/migoinfer"
	"go.uber.org/zap"
)

// newLogger returns a new logger with default options.
func newLogger() *migoinfer.Logger {
	color.NoColor = true
	l, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Cannot create new logger:", err)
	}
	return &migoinfer.Logger{SugaredLogger: l.Sugar()}
}

// newFileLogger returns a new logger and also writes the log output to files.
func newFileLogger(files ...string) *migoinfer.Logger {
	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = append(cfg.OutputPaths, files...)
	l, err := cfg.Build()
	if err != nil {
		log.Fatal("Cannot create new logger:", err)
	}
	return &migoinfer.Logger{SugaredLogger: l.Sugar()}
}

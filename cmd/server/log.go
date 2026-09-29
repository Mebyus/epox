package main

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/mebyus/epox/internal/froll"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func newLogger(cfg *LogConfig) (*zap.Logger, io.Closer, error) {
	path := filepath.Clean(cfg.File)
	if path == "" || path == "." || strings.Contains(path, "..") {
		panic(fmt.Sprintf("bad log file path \"%s\"", path))
	}

	roller, err := froll.NewRoller(&froll.Config{
		Path:    path,
		MaxSize: cfg.MaxSize,
		MaxNum:  cfg.MaxNum,
	})
	if err != nil {
		return nil, nil, err
	}

	core := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), roller, zap.DebugLevel)
	lg := zap.New(core)
	return lg, roller, nil
}

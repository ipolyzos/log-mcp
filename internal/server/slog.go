package server

import (
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	pkgLog     *slog.Logger
	pkgLogOnce sync.Once
)

func logger() *slog.Logger {
	pkgLogOnce.Do(func() {
		h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevelFromEnv()})
		pkgLog = slog.New(h)
	})
	return pkgLog
}

func logLevelFromEnv() slog.Level {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LOG_MCP_LOG_LEVEL"))) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

package server

import (
	"log/slog"
	"testing"
)

func TestLogLevelFromEnv(t *testing.T) {
	cases := []struct {
		env  string
		want slog.Level
	}{
		{"", slog.LevelInfo},
		{"info", slog.LevelInfo},
		{"DEBUG", slog.LevelDebug},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
	}
	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			t.Setenv("LOG_MCP_LOG_LEVEL", tc.env)
			if g := logLevelFromEnv(); g != tc.want {
				t.Fatalf("got %v want %v", g, tc.want)
			}
		})
	}
}

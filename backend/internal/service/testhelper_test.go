package service

import (
	"log/slog"
	"os"

	"shorturl/internal/config"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func testConfig() *config.Config {
	return &config.Config{
		BaseURL:      "http://localhost:8080",
		MaxURLsUser:  1000,
		BlockedHosts: map[string]struct{}{},
	}
}

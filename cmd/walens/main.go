package main

import (
	"log/slog"
	"os"

	"github.com/walens/walens/internal/app"
	"github.com/walens/walens/internal/config"
)

func main() {
	cfg := config.Load()

	if err := run(cfg); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config) error {
	application := app.New(cfg)
	return application.Start()
}

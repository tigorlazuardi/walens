package main

import (
	"log/slog"
	"os"

	"github.com/walens/walens/internal/app"
	"github.com/walens/walens/internal/config"
)

func main() {
	// Load dotenv files before config to allow .env overrides.
	// .dev.env loads first without overriding existing env vars.
	// .env loads second and overrides earlier values (including .dev.env).
	if err := config.LoadDotenv(); err != nil {
		slog.Error("dotenv load error", "error", err)
		os.Exit(1)
	}

	// Handle subcommands
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "openapi-yaml":
			openapiYAML()
			return
		}
	}

	// Default: run the application
	cfg := config.Load()

	if err := run(cfg); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

func openapiYAML() {
	cfg := config.Load()
	application := app.New(cfg)

	yamlBytes, err := application.OpenAPIYAML()
	if err != nil {
		slog.Error("failed to generate OpenAPI YAML", "error", err)
		os.Exit(1)
	}

	os.Stdout.Write(yamlBytes)
}

func run(cfg *config.Config) error {
	application := app.New(cfg)
	return application.Start()
}

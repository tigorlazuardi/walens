package config

import "testing"

func TestApplyPersistedConfig(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			Host:     "0.0.0.0",
			Port:     8080,
			BasePath: "/walens",
		},
		Database: DatabaseConfig{
			Path: "./data/walens.db",
		},
		Auth: AuthConfig{
			Enabled:  true,
			Username: "admin",
			Password: "secret",
		},
		DataDir:  "./data",
		LogLevel: "info",
	}

	// Apply persisted config with different values
	cfg.ApplyPersistedConfig("/opt/walens/data", "debug")

	// Bootstrap-only fields should be preserved
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("expected Host '0.0.0.0' to be preserved, got: %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected Port 8080 to be preserved, got: %d", cfg.Server.Port)
	}
	if cfg.Server.BasePath != "/walens" {
		t.Errorf("expected BasePath '/walens' to be preserved, got: %q", cfg.Server.BasePath)
	}
	if cfg.Database.Path != "./data/walens.db" {
		t.Errorf("expected Database.Path './data/walens.db' to be preserved, got: %q", cfg.Database.Path)
	}
	if cfg.Auth.Enabled != true {
		t.Errorf("expected Auth.Enabled true to be preserved, got: %v", cfg.Auth.Enabled)
	}
	if cfg.Auth.Username != "admin" {
		t.Errorf("expected Auth.Username 'admin' to be preserved, got: %q", cfg.Auth.Username)
	}

	// Persisted fields should be updated
	if cfg.DataDir != "/opt/walens/data" {
		t.Errorf("expected DataDir '/opt/walens/data', got: %q", cfg.DataDir)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got: %q", cfg.LogLevel)
	}
}

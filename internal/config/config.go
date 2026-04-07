package config

import (
	"errors"
	"os"
	"strconv"
)

// ErrAuthEnabledButNotConfigured is returned when auth is enabled but required auth settings are empty.
var ErrAuthEnabledButNotConfigured = errors.New("auth is enabled but username, password, or cookie secret is not configured")

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	DataDir  string
	LogLevel string
}

type ServerConfig struct {
	Host     string
	Port     int
	BasePath string
}

type DatabaseConfig struct {
	Path string
}

type AuthConfig struct {
	Enabled      bool
	Username     string
	Password     string
	CookieSecure bool
	CookieSecret string
}

// Validate checks that if auth is enabled, username and password are non-empty.
func (c AuthConfig) Validate() error {
	if c.Enabled {
		if c.Username == "" || c.Password == "" || c.CookieSecret == "" {
			return ErrAuthEnabledButNotConfigured
		}
	}
	return nil
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:     getEnv("WALENS_HOST", "0.0.0.0"),
			Port:     getEnvInt("WALENS_PORT", 8080),
			BasePath: getEnv("WALENS_BASE_PATH", "/"),
		},
		Database: DatabaseConfig{
			Path: getEnv("WALENS_DB_PATH", "./data/walens.db"),
		},
		Auth: AuthConfig{
			Enabled:      getEnvBool("WALENS_AUTH_ENABLED", false),
			Username:     getEnv("WALENS_AUTH_USERNAME", ""),
			Password:     getEnv("WALENS_AUTH_PASSWORD", ""),
			CookieSecure: getEnvBool("WALENS_AUTH_COOKIE_SECURE", false),
			CookieSecret: getEnv("WALENS_AUTH_COOKIE_SECRET", ""),
		},
		DataDir:  getEnv("WALENS_DATA_DIR", "./data"),
		LogLevel: getEnv("WALENS_LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal
		}
	}
	return defaultVal
}

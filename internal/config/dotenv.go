package config

import (
	"errors"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

// ErrDotenvLoadFailed is returned when a dotenv file exists but fails to load.
var ErrDotenvLoadFailed = errors.New("failed to load dotenv file")

// LoadDotenv loads environment variables from .dev.env first, then .env.
// Later files override earlier ones. Missing files are skipped.
func LoadDotenv() error {
	return loadDotenvWithSlog(nil)
}

// loadDotenvWithSlog loads dotenv files with optional slogger for warnings.
// If slogger is nil, warnings are logged via log/slog.
// .dev.env is loaded first without overriding existing env vars.
// .env is loaded second and overrides earlier values (including those from .dev.env).
func loadDotenvWithSlog(slogger *slog.Logger) error {
	// .dev.env - Load only, does not override existing env vars
	if err := loadDotenvFile(".dev.env", false, slogger); err != nil {
		return err
	}
	// .env - Overload, overrides existing env vars (including those from .dev.env)
	if err := loadDotenvFile(".env", true, slogger); err != nil {
		return err
	}
	return nil
}

func loadDotenvFile(file string, overload bool, slogger *slog.Logger) error {
	// Check if file exists before attempting to load
	if _, err := os.Stat(file); os.IsNotExist(err) {
		// File does not exist - skip silently
		return nil
	}

	// File exists (or stat failed for other reasons), try to load
	var err error
	if overload {
		err = godotenv.Overload(file)
	} else {
		err = godotenv.Load(file)
	}
	if err != nil {
		msg := "dotenv file '" + file + "' exists but failed to load: " + err.Error()
		if slogger != nil {
			slogger.Warn(msg)
		} else {
			slog.Warn(msg)
		}
		return errors.Join(ErrDotenvLoadFailed, err)
	}

	msg := "loaded dotenv file: " + file
	if slogger != nil {
		slogger.Info(msg)
	} else {
		slog.Info(msg)
	}
	return nil
}

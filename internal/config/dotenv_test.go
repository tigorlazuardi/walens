package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotenv(t *testing.T) {
	// Create a temp directory for test files
	tmpDir := t.TempDir()

	// Save current working directory and restore after test
	origCWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(origCWD)

	// Change to temp directory so dotenv files are created there
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	t.Run("missing files are skipped", func(t *testing.T) {
		// Ensure no dotenv files exist
		os.Remove(".dev.env")
		os.Remove(".env")
		t.Cleanup(func() {
			os.Remove(".dev.env")
			os.Remove(".env")
		})

		// Should not error when files don't exist
		err := LoadDotenv()
		if err != nil {
			t.Errorf("expected no error for missing files, got: %v", err)
		}
	})

	t.Run("dev env loaded first then env overrides", func(t *testing.T) {
		// Create .dev.env with TEST_KEY_DEV
		devEnv := `TEST_KEY_DEV=dev_value
TEST_KEY_COMMON=from_dev`
		if err := os.WriteFile(".dev.env", []byte(devEnv), 0644); err != nil {
			t.Fatalf("failed to write .dev.env: %v", err)
		}
		t.Cleanup(func() {
			os.Remove(".dev.env")
			os.Remove(".env")
		})

		// Create .env with TEST_KEY_ENV and override TEST_KEY_COMMON
		envContent := `TEST_KEY_ENV=env_value
TEST_KEY_COMMON=from_env`
		if err := os.WriteFile(".env", []byte(envContent), 0644); err != nil {
			t.Fatalf("failed to write .env: %v", err)
		}

		// Clear any existing TEST_KEY_* vars
		os.Unsetenv("TEST_KEY_DEV")
		os.Unsetenv("TEST_KEY_ENV")
		os.Unsetenv("TEST_KEY_COMMON")

		err := LoadDotenv()
		if err != nil {
			t.Fatalf("LoadDotenv failed: %v", err)
		}

		// .dev.env loaded first
		if got := os.Getenv("TEST_KEY_DEV"); got != "dev_value" {
			t.Errorf("expected TEST_KEY_DEV='dev_value', got: %q", got)
		}

		// .env loaded second and overrides
		if got := os.Getenv("TEST_KEY_ENV"); got != "env_value" {
			t.Errorf("expected TEST_KEY_ENV='env_value', got: %q", got)
		}

		// TEST_KEY_COMMON should be overridden by .env
		if got := os.Getenv("TEST_KEY_COMMON"); got != "from_env" {
			t.Errorf("expected TEST_KEY_COMMON='from_env', got: %q", got)
		}
	})

	t.Run("env file can override dev env values", func(t *testing.T) {
		// Create .dev.env
		devEnv := `OVERRIDE_TEST=dev`
		if err := os.WriteFile(".dev.env", []byte(devEnv), 0644); err != nil {
			t.Fatalf("failed to write .dev.env: %v", err)
		}
		t.Cleanup(func() {
			os.Remove(".dev.env")
			os.Remove(".env")
		})

		// Create .env that overrides
		envContent := `OVERRIDE_TEST=env`
		if err := os.WriteFile(".env", []byte(envContent), 0644); err != nil {
			t.Fatalf("failed to write .env: %v", err)
		}

		os.Unsetenv("OVERRIDE_TEST")

		err := LoadDotenv()
		if err != nil {
			t.Fatalf("LoadDotenv failed: %v", err)
		}

		// .env should override .dev.env
		if got := os.Getenv("OVERRIDE_TEST"); got != "env" {
			t.Errorf("expected OVERRIDE_TEST='env', got: %q", got)
		}
	})
}

func TestLoadDotenvFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.env")

	t.Run("missing file returns nil", func(t *testing.T) {
		nonExistent := filepath.Join(tmpDir, "nonexistent.env")
		err := loadDotenvFile(nonExistent, false, nil)
		if err != nil {
			t.Errorf("expected nil error for missing file, got: %v", err)
		}
	})

	t.Run("existing file loads successfully", func(t *testing.T) {
		content := `TEST_VAR=hello
ANOTHER_VAR=world`
		if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		os.Unsetenv("TEST_VAR")
		os.Unsetenv("ANOTHER_VAR")

		err := loadDotenvFile(testFile, false, nil)
		if err != nil {
			t.Errorf("expected successful load, got: %v", err)
		}

		if got := os.Getenv("TEST_VAR"); got != "hello" {
			t.Errorf("expected TEST_VAR='hello', got: %q", got)
		}
		if got := os.Getenv("ANOTHER_VAR"); got != "world" {
			t.Errorf("expected ANOTHER_VAR='world', got: %q", got)
		}
	})

	t.Run("invalid file returns error", func(t *testing.T) {
		// Create a file with invalid syntax (unclosed quote)
		invalidContent := `INVALID="unclosed`
		if err := os.WriteFile(testFile, []byte(invalidContent), 0644); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		err := loadDotenvFile(testFile, false, nil)
		if err == nil {
			t.Error("expected error for invalid dotenv file, got nil")
		}
		if err != nil && !errors.Is(err, ErrDotenvLoadFailed) {
			t.Errorf("expected ErrDotenvLoadFailed, got: %v", err)
		}
	})
}

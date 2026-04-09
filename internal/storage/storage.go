// Package storage provides file operations for the runner.
package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Config holds the storage configuration.
type Config struct {
	BaseDir string
}

// Service provides file operations for the runner.
type Service struct {
	cfg Config
}

// NewService creates a new storage service.
func NewService(cfg Config) *Service {
	return &Service{cfg: cfg}
}

// DownloadToTemp downloads a file from the given URL to a temporary location.
// Returns the path to the temp file, a cleanup function, and any error.
func (s *Service) DownloadToTemp(ctx context.Context, url string) (tempPath string, cleanup func(), err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "walens-*.tmp")
	if err != nil {
		return "", nil, fmt.Errorf("create temp file: %w", err)
	}
	tempPath = tmpFile.Name()

	cleanup = func() {
		os.Remove(tempPath)
	}

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		tmpFile.Close()
		cleanup()
		return "", nil, fmt.Errorf("copy body: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("close temp file: %w", err)
	}

	return tempPath, cleanup, nil
}

// MoveToCanonical moves a temp file to the canonical location.
// The canonical path is {base_dir}/images/{device_slug}/{unique_identifier}.{ext}
func (s *Service) MoveToCanonical(tempPath, deviceSlug, uniqueIdentifier, ext string) (canonicalPath string, err error) {
	canonicalDir := filepath.Join(s.cfg.BaseDir, "images", deviceSlug)
	if err := s.EnsureDir(canonicalDir); err != nil {
		return "", fmt.Errorf("ensure canonical dir: %w", err)
	}

	canonicalPath = filepath.Join(canonicalDir, fmt.Sprintf("%s.%s", uniqueIdentifier, ext))

	if err := os.Rename(tempPath, canonicalPath); err != nil {
		return "", fmt.Errorf("rename to canonical: %w", err)
	}

	return canonicalPath, nil
}

// CreateHardLink creates a hard link from source to target.
// Returns an error if hard links are not supported or not possible.
func (s *Service) CreateHardLink(sourcePath, targetPath string) error {
	if err := s.EnsureDir(filepath.Dir(targetPath)); err != nil {
		return fmt.Errorf("ensure target dir: %w", err)
	}

	if err := os.Link(sourcePath, targetPath); err != nil {
		return fmt.Errorf("create hard link: %w", err)
	}

	return nil
}

// CopyFile copies a file from source to target as a fallback.
func (s *Service) CopyFile(sourcePath, targetPath string) error {
	if err := s.EnsureDir(filepath.Dir(targetPath)); err != nil {
		return fmt.Errorf("ensure target dir: %w", err)
	}

	srcFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("create target: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy content: %w", err)
	}

	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}

	return nil
}

// FileExists checks if a file at the given path exists.
func (s *Service) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// EnsureDir ensures that a directory exists, creating it if necessary.
func (s *Service) EnsureDir(path string) error {
	if s.FileExists(path) {
		return nil
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("mkdir all: %w", err)
	}
	return nil
}

// BaseDir returns the configured base directory.
func (s *Service) BaseDir() string {
	return s.cfg.BaseDir
}

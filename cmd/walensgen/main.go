// Package main implements the walensgen command, Walens's Go-Jet
// generation orchestration tool using the generator API directly.
//
// Usage:
//
//	walensgen generate [flags]
//
// Flags:
//
//	-out string
//	      Output directory for generated code (default "./internal/db/generated")
//	-tempdb string
//	      Path for temporary SQLite DB used during codegen (default uses temp file)
//	-migrations string
//	      Path to migrations directory (default "./internal/db/migrations")
//
// The generate command:
//
//  1. Creates a temporary SQLite database file
//  2. Runs embedded migrations against it
//  3. Invokes Go-Jet generation with Walens customizations via API
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	jetGenerator "github.com/go-jet/jet/v2/generator/sqlite"
	"github.com/pressly/goose/v3"
	"github.com/walens/walens/internal/codegen"
	_ "modernc.org/sqlite"
)

// Default paths relative to repo root.
var (
	repoRoot          = findRepoRoot()
	defaultMigrations = filepath.Join(repoRoot, "internal", "db", "migrations")
	defaultOutput     = filepath.Join(repoRoot, "internal", "db", "generated")
)

func main() {
	if err := run(os.Args); err != nil {
		log.Fatalf("walensgen: %v", err)
	}
}

func run(args []string) error {
	if len(args) < 2 || args[1] != "generate" {
		return fmt.Errorf("usage: walensgen generate [flags]\nOnly 'generate' command is supported")
	}

	gen := &generateCmd{
		outDir:      defaultOutput,
		tempDBPath:  "", // Will be set to a temp file
		migrations:  defaultMigrations,
		packageName: "model",
	}

	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.StringVar(&gen.outDir, "out", gen.outDir, "output directory for generated code")
	fs.StringVar(&gen.tempDBPath, "tempdb", gen.tempDBPath, "path for temporary SQLite DB (default creates temp file)")
	fs.StringVar(&gen.migrations, "migrations", gen.migrations, "path to migrations directory")
	fs.StringVar(&gen.packageName, "package", gen.packageName, "Go package name for generated code")

	if err := fs.Parse(args[2:]); err != nil {
		return err
	}

	return gen.Run()
}

type generateCmd struct {
	outDir        string
	tempDBPath    string // If empty, a temp file is created
	migrations    string
	packageName   string
	createdTempDB bool // True if we created the temp DB (and should clean it up)
}

func (c *generateCmd) Run() error {
	ctx := context.Background()

	// Step 1: Create temp DB
	db, err := c.createTempDB(ctx)
	if err != nil {
		return fmt.Errorf("create temp DB: %w", err)
	}
	defer func() {
		db.Close()
		if c.createdTempDB && c.tempDBPath != ":memory:" {
			os.Remove(c.tempDBPath)
		}
	}()

	log.Printf("Created temp DB at %s", c.tempDBPath)

	// Step 2: Run migrations
	if err := c.runMigrations(ctx, db); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	log.Printf("Ran migrations from %s", c.migrations)

	// Step 3: Invoke Go-Jet generation with Walens customizations
	if err := c.invokeJetGeneration(ctx, db); err != nil {
		return fmt.Errorf("invoke Jet: %w", err)
	}

	log.Printf("Jet generation complete (output: %s)", c.outDir)

	return nil
}

func (c *generateCmd) createTempDB(ctx context.Context) (*sql.DB, error) {
	var dbPath string

	if c.tempDBPath != "" && c.tempDBPath != ":memory:" {
		// User-specified path
		dbPath = c.tempDBPath
	} else {
		// Create temp file
		f, err := os.CreateTemp("", "walens_codegen_*.db")
		if err != nil {
			return nil, fmt.Errorf("create temp file: %w", err)
		}
		dbPath = f.Name()
		f.Close()
		c.createdTempDB = true
	}

	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open temp DB: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping temp DB: %w", err)
	}

	c.tempDBPath = dbPath
	return db, nil
}

func (c *generateCmd) runMigrations(ctx context.Context, db *sql.DB) error {
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, os.DirFS(c.migrations))
	if err != nil {
		return fmt.Errorf("create goose provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("run goose up: %w", err)
	}

	return nil
}

func (c *generateCmd) invokeJetGeneration(ctx context.Context, db *sql.DB) error {
	// Ensure output directory exists
	if err := os.MkdirAll(c.outDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	// Clean output directory of existing generated files
	if err := c.cleanOutputDir(); err != nil {
		log.Printf("Warning: could not clean output dir: %v", err)
	}

	// Build Walens-customized template
	walensTmpl := codegen.BuildWalensTemplate(c.packageName)

	log.Printf("Invoking Go-Jet generator API (output: %s)", c.outDir)

	if err := jetGenerator.GenerateDB(db, c.outDir, walensTmpl); err != nil {
		return fmt.Errorf("jet GenerateDB: %w", err)
	}

	return nil
}

func (c *generateCmd) cleanOutputDir() error {
	entries, err := os.ReadDir(c.outDir)
	if err != nil {
		return err
	}

	// Don't remove hidden files or non-Go files
	for _, entry := range entries {
		if entry.IsDir() {
			// Clean subdirectories too (model/, table/)
			subDir := filepath.Join(c.outDir, entry.Name())
			if err := os.RemoveAll(subDir); err != nil {
				log.Printf("Warning: could not remove %s: %v", subDir, err)
			}
			continue
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		// Remove existing generated files
		if err := os.Remove(filepath.Join(c.outDir, name)); err != nil {
			log.Printf("Warning: could not remove %s: %v", name, err)
		}
	}
	return nil
}

// findRepoRoot attempts to find the repository root by looking for go.mod.
func findRepoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

// GetOutDir returns the configured output directory.
func GetOutDir() string {
	return defaultOutput
}

// GetMigrationsDir returns the configured migrations directory.
func GetMigrationsDir() string {
	return defaultMigrations
}

package database

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations applies every .sql file under ./migrations in filename
func RunMigrations() error {
	const migrationsDir = "./migrations"

	var migrationFiles []string

	err := filepath.WalkDir(migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".sql") {
			migrationFiles = append(migrationFiles, path)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan migrations: %w", err)
	}

	sort.Strings(migrationFiles)

	// Run migrations
	for _, file := range migrationFiles {
		slog.Info("running migration", "file", file)

		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}

		if err := DB.Exec(string(sqlBytes)).Error; err != nil {
			return fmt.Errorf("migration %s failed: %w", file, err)
		}

		slog.Info("migration completed", "file", file)
	}

	return nil
}
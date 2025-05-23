package commands

import (
	"fmt"
	"go-ldap-sso/config"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func _MigrateUp(cfg *config.Config) error {
	dbURL := cfg.GetDBUrl()
	m, err := migrate.New(
		"file://"+filepath.Join("db/migrations"),
		dbURL,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	log.Println("✅ Migrations applied successfully")
	return nil
}

func MigrateUp(cfg *config.Config) error {
	// Get current working directory (should be project root)
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	migrationsPath := "file://" + filepath.Join(wd, "db", "migrations")

	// Print path for debug
	log.Println("📂 Using migrations from:", migrationsPath)

	dbURL := cfg.GetDBUrl()
	m, err := migrate.New(
		migrationsPath,
		dbURL,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	log.Println("✅ Migrations applied successfully")
	return nil
}

func MigrateDown(cfg *config.Config) error {
	dbURL := cfg.GetDBUrl()

	m, err := migrate.New(
		"file://"+filepath.Join("db/migrations"),
		dbURL,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migrator: %w", err)
	}
	defer m.Close()

	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}

	log.Println("✅ Migrations rolled back successfully")
	return nil
}

func CreateMigration(name string) error {
	if name == "" {
		return fmt.Errorf("migration name cannot be empty")
	}

	if err := os.MkdirAll("db/migrations", 0755); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	files, err := filepath.Glob("db/migrations/*.up.sql")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	version := 1
	if len(files) > 0 {
		lastFile := files[len(files)-1]
		_, err := fmt.Sscanf(filepath.Base(lastFile), "%d", &version)
		if err == nil {
			version++
		}
	}

	upFile := fmt.Sprintf("db/migrations/%06d_%s.up.sql", version, name)
	downFile := fmt.Sprintf("db/migrations/%06d_%s.down.sql", version, name)

	upContent := `-- Add migration script here
`
	downContent := `-- Add rollback script here
`

	if err := os.WriteFile(upFile, []byte(upContent), 0644); err != nil {
		return fmt.Errorf("failed to create up migration: %w", err)
	}

	if err := os.WriteFile(downFile, []byte(downContent), 0644); err != nil {
		os.Remove(upFile)
		return fmt.Errorf("failed to create down migration: %w", err)
	}

	log.Printf("✅ Created migrations:\n- %s\n- %s\n", upFile, downFile)
	return nil
}

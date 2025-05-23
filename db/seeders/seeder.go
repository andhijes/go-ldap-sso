package seeders

import (
	"context"
	"fmt"
	"go-ldap-sso/db/seedhistory"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FileSeeder struct {
	Name     string
	FilePath string
}

type Seeder interface {
	GetName() string
	Seed(ctx context.Context, tx pgx.Tx) error
}

func (fs FileSeeder) GetName() string {
	return fs.Name
}

func (fs FileSeeder) Seed(ctx context.Context, tx pgx.Tx) error {
	content, err := os.ReadFile(fs.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read seeder file: %w", err)
	}

	if _, err := tx.Exec(ctx, string(content)); err != nil {
		return fmt.Errorf("failed to execute seeder SQL: %w", err)
	}

	return nil
}

type SeederRunner struct {
	repo       *seedhistory.Repository
	pool       *pgxpool.Pool
	seedersDir string
}

func NewSeederRunner(pool *pgxpool.Pool, seedersDir string) *SeederRunner {
	return &SeederRunner{
		repo:       seedhistory.NewRepository(pool),
		pool:       pool,
		seedersDir: seedersDir,
	}
}

func (r *SeederRunner) LoadSeedersFromFiles() ([]Seeder, error) {
	var seeders []Seeder

	files, err := os.ReadDir(r.seedersDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read seeders directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		// Format nama: {timestamp}_{name}.sql
		parts := strings.Split(strings.TrimSuffix(file.Name(), ".sql"), "_")
		if len(parts) < 2 {
			continue
		}

		name := strings.Join(parts[1:], "_")
		seeders = append(seeders, FileSeeder{
			Name:     name,
			FilePath: filepath.Join(r.seedersDir, file.Name()),
		})
	}

	return seeders, nil
}

func (r *SeederRunner) Run(ctx context.Context) error {
	if err := r.repo.Initialize(ctx); err != nil {
		return fmt.Errorf("failed to initialize seed history: %w", err)
	}

	seeders, err := r.LoadSeedersFromFiles()
	if err != nil {
		return fmt.Errorf("failed to load seeders: %w", err)
	}

	if len(seeders) == 0 {
		log.Println("ℹ️ No seeders found")
		return nil
	}

	batch, err := r.repo.GetLastBatch(ctx)
	if err != nil {
		return fmt.Errorf("failed to get last batch: %w", err)
	}
	batch++

	seededNames, err := r.repo.GetSeededNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to get seeded names: %w", err)
	}

	for _, seeder := range seeders {
		if _, exists := seededNames[seeder.GetName()]; exists {
			log.Printf("ℹ️ Seeder %s already applied, skipping", seeder.GetName())
			continue
		}

		tx, err := r.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		log.Printf("🏃 Running seeder: %s", seeder.GetName())
		if err := seeder.Seed(ctx, tx); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("seeder %s failed: %w", seeder.GetName(), err)
		}

		if err := r.repo.RecordSeed(ctx, tx, seeder.GetName(), batch); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to record seed history: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		log.Printf("✅ Seeder %s completed", seeder.GetName())
	}

	log.Printf("🎉 All seeders completed (batch %d)", batch)
	return nil
}

package seedhistory

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Initialize(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS seed_history (
            id SERIAL PRIMARY KEY,
            name VARCHAR(255) NOT NULL,
            batch INTEGER NOT NULL,
            applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
	return err
}

func (r *Repository) GetSeededNames(ctx context.Context) (map[string]bool, error) {
	rows, err := r.pool.Query(ctx, "SELECT name FROM seed_history")
	if err != nil {
		return nil, fmt.Errorf("failed to query seeded names: %w", err)
	}
	defer rows.Close()

	seeded := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan name: %w", err)
		}
		seeded[name] = true
	}
	return seeded, nil
}

func (r *Repository) RecordSeed(ctx context.Context, tx pgx.Tx, name string, batch int) error {
	_, err := tx.Exec(
		ctx,
		"INSERT INTO seed_history (name, batch) VALUES ($1, $2)",
		name, batch,
	)
	return err
}

func (r *Repository) GetLastBatch(ctx context.Context) (int, error) {
	var batch int
	err := r.pool.QueryRow(
		ctx,
		"SELECT COALESCE(MAX(batch), 0) FROM seed_history",
	).Scan(&batch)
	return batch, err
}

func (r *Repository) GetHistory(ctx context.Context) ([]SeedHistory, error) {
	rows, err := r.pool.Query(
		ctx,
		"SELECT id, name, batch, applied_at FROM seed_history ORDER BY batch DESC, applied_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []SeedHistory
	for rows.Next() {
		var h SeedHistory
		if err := rows.Scan(&h.ID, &h.Name, &h.Batch, &h.AppliedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
}

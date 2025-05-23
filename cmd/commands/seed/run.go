package seed

import (
	"context"
	"fmt"
	"go-ldap-sso/db/seeders"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Run(ctx context.Context, pool *pgxpool.Pool) error {
	seedersDir := "seeders"

	runner := seeders.NewSeederRunner(pool, seedersDir)
	if err := runner.Run(ctx); err != nil {
		return fmt.Errorf("seed failed: %w", err)
	}

	return nil
}

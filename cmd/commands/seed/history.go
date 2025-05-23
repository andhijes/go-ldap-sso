package seed

import (
	"context"
	"fmt"
	"go-ldap-sso/db/seedhistory"

	"github.com/jackc/pgx/v5/pgxpool"
)

func History(ctx context.Context, pool *pgxpool.Pool) error {
	repo := seedhistory.NewRepository(pool)

	history, err := repo.GetHistory(ctx)
	if err != nil {
		return fmt.Errorf("failed to get history: %w", err)
	}

	fmt.Println("Seed History:")
	fmt.Println("----------------------------------------")
	fmt.Printf("%-20s | %-6s | %-19s\n", "Name", "Batch", "Applied At")
	fmt.Println("----------------------------------------")

	for _, h := range history {
		fmt.Printf("%-20s | %-6d | %-19s\n", h.Name, h.Batch, h.AppliedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

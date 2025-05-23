package db

import (
	"context"
	"go-ldap-sso/config"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	Pool *pgxpool.Pool
}

func NewDatabase(config *config.Config) *Database {
	url := config.GetDBUrl()

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		log.Fatalf("❌ Unable to parse database config: %v", err)
	}

	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		log.Fatalf("❌ Unable to create DB pool: %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("❌ Unable to ping DB: %v", err)
	}

	log.Println("✅ Connected to the database")

	return &Database{Pool: pool}
}

func (d *Database) Close() {
	if d.Pool != nil {
		d.Pool.Close()
		log.Println("✅ Database connection closed")
	}
}

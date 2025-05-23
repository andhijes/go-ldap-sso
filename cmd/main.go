package main

import (
	"context"
	"go-ldap-sso/cmd/commands"
	"go-ldap-sso/cmd/commands/seed"
	"go-ldap-sso/config"
	"go-ldap-sso/db"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	app := &cli.App{
		Name:  "goLDAPSso",
		Usage: "Go LDAP SSO",
		Commands: []*cli.Command{
			{
				Name:  "api",
				Usage: "Run the API server",
				Action: func(c *cli.Context) error {
					return commands.RunAPI(cfg)
				},
			},
			{
				Name:  "idp",
				Usage: "Run the API server",
				Action: func(c *cli.Context) error {
					return commands.RunIDP(cfg)
				},
			},
			{
				Name:  "migrate",
				Usage: "Database migrations",
				Subcommands: []*cli.Command{
					{
						Name:  "up",
						Usage: "Apply all pending migrations",
						Action: func(c *cli.Context) error {
							return commands.MigrateUp(cfg)
						},
					},
					{
						Name:  "down",
						Usage: "Rollback last migration",
						Action: func(c *cli.Context) error {
							return commands.MigrateDown(cfg)
						},
					},
					{
						Name:  "create",
						Usage: "Create new migration file",
						Action: func(c *cli.Context) error {
							if c.NArg() < 1 {
								return cli.Exit("Error: Migration name is required", 1)
							}
							return commands.CreateMigration(c.Args().First())
						},
						UsageText: "create <migration_name>",
					},
				},
			},
			{
				Name:  "seed",
				Usage: "Database seeding operations",
				Subcommands: []*cli.Command{
					{
						Name:  "create",
						Usage: "Create new seeder file",
						Action: func(c *cli.Context) error {
							if c.NArg() < 1 {
								return cli.Exit("Seeder name is required", 1)
							}
							return seed.Create(c.Args().First())
						},
					},
					{
						Name:  "run",
						Usage: "Run all pending seeders",
						Action: func(c *cli.Context) error {
							ctx := context.Background()
							dbConn := db.NewDatabase(cfg)
							defer dbConn.Close()

							return seed.Run(ctx, dbConn.Pool)
						},
					},
					{
						Name:  "history",
						Usage: "Show seeding history",
						Action: func(c *cli.Context) error {
							ctx := context.Background()
							dbConn := db.NewDatabase(cfg)
							defer dbConn.Close()

							return seed.History(ctx, dbConn.Pool)
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

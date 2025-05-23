package seed

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func Create(name string) error {
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.sql", timestamp, name)
	path := filepath.Join("seeders", filename)

	if err := os.MkdirAll("seeders", 0755); err != nil {
		return fmt.Errorf("failed to create seeders directory: %w", err)
	}

	content := fmt.Sprintf(`-- Seeder: %s
-- Timestamp: %s

INSERT INTO your_table (columns) VALUES (values);
-- Add more seed data as needed
`, name, time.Now().Format(time.RFC3339))

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create seeder file: %w", err)
	}

	fmt.Printf("Created seeder: %s\n", path)
	return nil
}

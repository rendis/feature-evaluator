package postgres

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"sort"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// RunMigrations applies embedded SQL migrations in filename order.
func RunMigrations(ctx context.Context, client *Client) error {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		files = append(files, entry.Name())
	}
	sort.Strings(files)

	if _, err := client.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	for _, file := range files {
		var applied bool
		if err := client.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)`, file).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", file, err)
		}
		if applied {
			continue
		}

		content, err := migrationFS.ReadFile("migrations/" + file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}

		if err := client.WithinTx(ctx, func(txCtx context.Context) error {
			if _, err := client.db(txCtx).Exec(txCtx, string(content)); err != nil {
				return fmt.Errorf("exec migration %s: %w", file, err)
			}
			if _, err := client.db(txCtx).Exec(
				txCtx,
				`INSERT INTO schema_migrations (filename, applied_at) VALUES ($1, NOW())`,
				file,
			); err != nil {
				return fmt.Errorf("record migration %s: %w", file, err)
			}
			return nil
		}); err != nil {
			return err
		}

		slog.Info("applied PostgreSQL migration", "file", file)
	}

	return nil
}

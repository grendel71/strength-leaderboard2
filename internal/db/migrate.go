package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ApplyMigrations applies any new migrations/*.up.sql files.
//
// It keeps track of applied migrations in the schema_migrations table.
func ApplyMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}

	_, err := pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
	filename TEXT PRIMARY KEY,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied := map[string]struct{}{}
	rows, err := pool.Query(ctx, "SELECT filename FROM schema_migrations")
	if err != nil {
		return fmt.Errorf("list schema_migrations: %w", err)
	}
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			rows.Close()
			return fmt.Errorf("scan schema_migrations: %w", scanErr)
		}
		applied[name] = struct{}{}
	}
	rows.Close()

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir %q: %w", migrationsDir, err)
	}

	var migrationFiles []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".up.sql") {
			migrationFiles = append(migrationFiles, name)
		}
	}
	sort.Strings(migrationFiles)

	for _, name := range migrationFiles {
		if _, ok := applied[name]; ok {
			continue
		}

		// Bootstrap: if the DB was initialized manually before we introduced
		// schema_migrations, skip the non-idempotent initial migration.
		if name == "001_initial.up.sql" {
			var exists bool
			err := pool.QueryRow(ctx, "SELECT to_regclass('public.athletes') IS NOT NULL").Scan(&exists)
			if err != nil {
				return fmt.Errorf("check athletes table: %w", err)
			}
			if exists {
				if _, err := pool.Exec(ctx, "INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING", name); err != nil {
					return fmt.Errorf("record schema_migrations %q: %w", name, err)
				}
				continue
			}
		}

		path := filepath.Join(migrationsDir, name)
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %q: %w", path, err)
		}

		// Skip empty or whitespace-only files.
		if strings.TrimSpace(string(sqlBytes)) == "" {
			continue
		}

		err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			if _, execErr := tx.Exec(ctx, string(sqlBytes)); execErr != nil {
				return fmt.Errorf("exec migration %q: %w", name, execErr)
			}
			if _, execErr := tx.Exec(ctx, "INSERT INTO schema_migrations (filename) VALUES ($1)", name); execErr != nil {
				return fmt.Errorf("record schema_migrations %q: %w", name, execErr)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

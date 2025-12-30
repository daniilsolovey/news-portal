package db

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	"github.com/pressly/goose/v3"
)

func withTx(t *testing.T) (*pg.Tx, context.Context, *Repository) {
	t.Helper()
	ctx := context.Background()

	tx, err := testDB.Begin()
	if err != nil {
		t.Fatalf("failed to begin transaction: %v", err)
	}

	t.Cleanup(func() {
		if err := tx.Rollback(); err != nil {
			t.Errorf("failed to rollback transaction: %v", err)
		}
	})

	repo := New(tx, testLogger)
	return tx, ctx, repo
}

func resetPublicSchema(ctx context.Context, db *pg.DB) error {
	_, err := db.ExecContext(ctx, `DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`)
	if err != nil {
		return fmt.Errorf("reset public schema: %w", err)
	}
	return nil
}

func runMigrations(ctx context.Context) error {
	config, err := pgx.ParseConnectionString(testDBURL)
	if err != nil {
		return fmt.Errorf("parse connection string: %w", err)
	}

	sqldb := stdlib.OpenDB(config)
	defer sqldb.Close()

	if err := sqldb.PingContext(ctx); err != nil {
		return fmt.Errorf("ping test db: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return fmt.Errorf("no migration files found in %s", migrationsDir)
	}

	if err := goose.UpContext(ctx, sqldb, migrationsDir); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}

func ensureTablesExist(ctx context.Context, db *pg.DB, tables []string) error {
	for _, tbl := range tables {
		var exists bool
		_, err := db.QueryOneContext(ctx, pg.Scan(&exists), `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = ?
			)`, tbl)
		if err != nil {
			return fmt.Errorf("check table %s exists: %w", tbl, err)
		}
		if !exists {
			return fmt.Errorf("table %q does not exist after migrations", tbl)
		}
	}
	return nil
}

func loadTestData(ctx context.Context, db *pg.DB) error {
	_, err := db.ExecContext(ctx, `
		TRUNCATE TABLE "news", "tags", "categories", "statuses" RESTART IDENTITY CASCADE;
	`)
	if err != nil {
		return fmt.Errorf("truncate tables: %w", err)
	}

	_, err = db.ExecContext(ctx, `INSERT INTO "statuses" ("statusId") VALUES (1), (2), (3) ON CONFLICT DO NOTHING`)
	if err != nil {
		return fmt.Errorf("insert statuses: %w", err)
	}

	categories := []Category{
		{Title: "Technology", OrderNumber: 1, StatusID: statusPublished},
		{Title: "Sports", OrderNumber: 2, StatusID: statusPublished},
		{Title: "Politics", OrderNumber: 3, StatusID: statusPublished},
		{Title: "Economy", OrderNumber: 4, StatusID: statusPublished},
		{Title: "Culture", OrderNumber: 5, StatusID: statusPublished},
	}
	for i := range categories {
		if _, err := db.ModelContext(ctx, &categories[i]).Insert(); err != nil {
			return fmt.Errorf("insert category %q: %w", categories[i].Title, err)
		}
	}

	tags := []Tag{
		{Title: "Important", StatusID: statusPublished},
		{Title: "Hot", StatusID: statusPublished},
		{Title: "Analytics", StatusID: statusPublished},
		{Title: "Interview", StatusID: statusPublished},
		{Title: "Report", StatusID: statusPublished},
	}
	for i := range tags {
		if _, err := db.ModelContext(ctx, &tags[i]).Insert(); err != nil {
			return fmt.Errorf("insert tag %q: %w", tags[i].Title, err)
		}
	}

	content1 := "Artificial intelligence continues to evolve rapidly. New machine learning models show impressive results."
	content2 := "Quantum computers promise to revolutionize computing technology. Scientists have made significant progress."
	content3 := "The World Cup has concluded. Teams showed high level of play."
	content4 := "New world records were set at the Olympic Games. Athletes demonstrate incredible results."
	content5 := "An international summit concluded, discussing important global policy issues."
	content6 := "Experts analyze the current situation in financial markets. Certain trends are noted."
	content7 := "An international film festival concluded. The jury determined winners in various categories."

	newsItems := []News{
		{
			CategoryID:  1,
			Title:       "AI Breakthrough in Machine Learning",
			Content:     &content1,
			Author:      "John Doe",
			PublishedAt: baseTime.Add(-0 * 24 * time.Hour),
			TagIDs:      []int{1, 2},
			StatusID:    statusPublished,
		},
		{
			CategoryID:  1,
			Title:       "Quantum Computers: Future of Computing",
			Content:     &content2,
			Author:      "Jane Smith",
			PublishedAt: baseTime.Add(-1 * 24 * time.Hour),
			TagIDs:      []int{1, 3},
			StatusID:    statusPublished,
		},
		{
			CategoryID:  2,
			Title:       "World Cup Finals: Results",
			Content:     &content3,
			Author:      "Bob Johnson",
			PublishedAt: baseTime.Add(-2 * 24 * time.Hour),
			TagIDs:      []int{1, 2},
			StatusID:    statusPublished,
		},
		{
			CategoryID:  2,
			Title:       "Olympic Games: New Records",
			Content:     &content4,
			Author:      "Alice Brown",
			PublishedAt: baseTime.Add(-3 * 24 * time.Hour),
			TagIDs:      []int{1, 5},
			StatusID:    statusPublished,
		},
		{
			CategoryID:  3,
			Title:       "International Summit: Negotiation Results",
			Content:     &content5,
			Author:      "Charlie Wilson",
			PublishedAt: baseTime.Add(-4 * 24 * time.Hour),
			TagIDs:      []int{1, 3},
			StatusID:    statusPublished,
		},
		{
			CategoryID:  4,
			Title:       "Financial Markets: Situation Analysis",
			Content:     &content6,
			Author:      "Diana Davis",
			PublishedAt: baseTime.Add(-5 * 24 * time.Hour),
			TagIDs:      []int{1, 3},
			StatusID:    statusPublished,
		},
		{
			CategoryID:  5,
			Title:       "Film Festival: Award Ceremony",
			Content:     &content7,
			Author:      "Edward Miller",
			PublishedAt: baseTime.Add(-6 * 24 * time.Hour),
			TagIDs:      []int{1, 2},
			StatusID:    statusPublished,
		},
	}

	for i := range newsItems {
		if _, err := db.ModelContext(ctx, &newsItems[i]).Insert(); err != nil {
			return fmt.Errorf("insert news %q: %w", newsItems[i].Title, err)
		}
	}

	return nil
}

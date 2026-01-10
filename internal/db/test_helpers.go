package db

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/jackc/pgx"
	"github.com/jackc/pgx/stdlib"
	"github.com/pressly/goose/v3"
)

const (
	// TestDBURL is the connection string for the test database
	TestDBURL = "postgres://test_user:test_password@localhost:5433/news_portal_test?sslmode=disable"
	// MigrationsDir is the directory containing test migrations
	MigrationsDir = "../../docs/patches/integrationtests"
)

var (
	// BaseTime is the base time used for test data
	BaseTime = time.Date(2024, 1, 14, 12, 0, 0, 0, time.UTC)
)

// ResetPublicSchema drops and recreates the public schema
func ResetPublicSchema(ctx context.Context, database *pg.DB) error {
	_, err := database.ExecContext(ctx, `DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`)
	if err != nil {
		return fmt.Errorf("reset public schema: %w", err)
	}
	return nil
}

// RunMigrations runs database migrations from the migrations directory
func RunMigrations(ctx context.Context, migrationsDir string) error {
	config, err := pgx.ParseConnectionString(TestDBURL)
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

// EnsureTablesExist verifies that the specified tables exist in the database
func EnsureTablesExist(ctx context.Context, database *pg.DB, tables []string) error {
	for _, tbl := range tables {
		var exists bool
		_, err := database.QueryOneContext(ctx, pg.Scan(&exists), `
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

// LoadTestData loads test data into the database
func LoadTestData(ctx context.Context, database *pg.DB) error {
	_, err := database.ExecContext(ctx, `
		TRUNCATE TABLE "news", "tags", "categories", "statuses" RESTART IDENTITY CASCADE;
	`)
	if err != nil {
		return fmt.Errorf("truncate tables: %w", err)
	}

	_, err = database.ExecContext(ctx, `INSERT INTO "statuses" ("statusId") VALUES (1), (2), (3) ON CONFLICT DO NOTHING`)
	if err != nil {
		return fmt.Errorf("insert statuses: %w", err)
	}

	categories := []Category{
		{Title: "Technology", OrderNumber: 1, StatusID: 1},
		{Title: "Sports", OrderNumber: 2, StatusID: 1},
		{Title: "Politics", OrderNumber: 3, StatusID: 1},
		{Title: "Economy", OrderNumber: 4, StatusID: 1},
		{Title: "Culture", OrderNumber: 5, StatusID: 1},
	}
	for i := range categories {
		if _, err := database.ModelContext(ctx, &categories[i]).Insert(); err != nil {
			return fmt.Errorf("insert category %q: %w", categories[i].Title, err)
		}
	}

	tags := []Tag{
		{Title: "Important", StatusID: 1},
		{Title: "Hot", StatusID: 1},
		{Title: "Analytics", StatusID: 1},
		{Title: "Interview", StatusID: 1},
		{Title: "Report", StatusID: 1},
	}
	for i := range tags {
		if _, err := database.ModelContext(ctx, &tags[i]).Insert(); err != nil {
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
			PublishedAt: BaseTime.Add(-0 * 24 * time.Hour),
			TagIDs:      []int{1, 2},
			StatusID:    1,
		},
		{
			CategoryID:  1,
			Title:       "Quantum Computers: Future of Computing",
			Content:     &content2,
			Author:      "Jane Smith",
			PublishedAt: BaseTime.Add(-1 * 24 * time.Hour),
			TagIDs:      []int{1, 3},
			StatusID:    1,
		},
		{
			CategoryID:  2,
			Title:       "World Cup Finals: Results",
			Content:     &content3,
			Author:      "Bob Johnson",
			PublishedAt: BaseTime.Add(-2 * 24 * time.Hour),
			TagIDs:      []int{1, 2},
			StatusID:    1,
		},
		{
			CategoryID:  2,
			Title:       "Olympic Games: New Records",
			Content:     &content4,
			Author:      "Alice Brown",
			PublishedAt: BaseTime.Add(-3 * 24 * time.Hour),
			TagIDs:      []int{1, 5},
			StatusID:    1,
		},
		{
			CategoryID:  3,
			Title:       "International Summit: Negotiation Results",
			Content:     &content5,
			Author:      "Charlie Wilson",
			PublishedAt: BaseTime.Add(-4 * 24 * time.Hour),
			TagIDs:      []int{1, 3},
			StatusID:    1,
		},
		{
			CategoryID:  4,
			Title:       "Financial Markets: Situation Analysis",
			Content:     &content6,
			Author:      "Diana Davis",
			PublishedAt: BaseTime.Add(-5 * 24 * time.Hour),
			TagIDs:      []int{1, 3},
			StatusID:    1,
		},
		{
			CategoryID:  5,
			Title:       "Film Festival: Award Ceremony",
			Content:     &content7,
			Author:      "Edward Miller",
			PublishedAt: BaseTime.Add(-6 * 24 * time.Hour),
			TagIDs:      []int{1, 2},
			StatusID:    1,
		},
	}

	for i := range newsItems {
		if _, err := database.ModelContext(ctx, &newsItems[i]).Insert(); err != nil {
			return fmt.Errorf("insert news %q: %w", newsItems[i].Title, err)
		}
	}

	return nil
}

// SetupTestDB initializes the test database connection and sets up the schema
func SetupTestDB() (*pg.DB, error) {
	ctx := context.Background()

	opt, err := pg.ParseURL(TestDBURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	database := pg.Connect(opt)

	if err := database.Ping(ctx); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("failed to connect to test database: %w", err)
	}

	if err := ResetPublicSchema(ctx, database); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("failed to reset schema: %w", err)
	}

	if err := RunMigrations(ctx, MigrationsDir); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	if err := EnsureTablesExist(ctx, database, []string{"statuses", "categories", "tags", "news"}); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("schema verification failed: %w", err)
	}

	if err := LoadTestData(ctx, database); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("failed to load test data: %w", err)
	}

	return database, nil
}

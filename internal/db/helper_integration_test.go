package db

import (
	"context"
	"testing"

	"github.com/go-pg/pg/v10"
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

	repo := New(tx)
	return tx, ctx, repo
}

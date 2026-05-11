package migrate

import (
	"context"
	_ "embed"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed sql/001_create_payments.sql
var schema string

// Run applies the schema. Safe to call on every startup — all statements use IF NOT EXISTS.
func Run(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, schema)
	return err
}

package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"stepik-payments-course/internal/payment"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func scan(row pgx.Row) (payment.Payment, error) {
	var p payment.Payment
	err := row.Scan(&p.ID, &p.OrderID, &p.Amount, &p.Currency, &p.Description, &p.Status, &p.PaymentURL, &p.ProviderPaymentID, &p.IdempotencyKey, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return payment.Payment{}, payment.ErrNotFound
	}
	return p, err
}

func (r *Repository) Create(ctx context.Context, p payment.Payment) (payment.Payment, error) {
	q := `INSERT INTO payments (id, order_id, amount, currency, description, status, payment_url, provider_payment_id, idempotency_key)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
RETURNING id, order_id, amount, currency, description, status, payment_url, provider_payment_id, idempotency_key, created_at, updated_at`
	return scan(r.db.QueryRow(ctx, q, p.ID, p.OrderID, p.Amount, p.Currency, p.Description, p.Status, p.PaymentURL, p.ProviderPaymentID, p.IdempotencyKey))
}

func (r *Repository) Get(ctx context.Context, id string) (payment.Payment, error) {
	q := `SELECT id, order_id, amount, currency, description, status, payment_url, provider_payment_id, idempotency_key, created_at, updated_at FROM payments WHERE id=$1`
	return scan(r.db.QueryRow(ctx, q, id))
}

func (r *Repository) FindByIdempotencyKey(ctx context.Context, key string) (payment.Payment, error) {
	q := `SELECT id, order_id, amount, currency, description, status, payment_url, provider_payment_id, idempotency_key, created_at, updated_at FROM payments WHERE idempotency_key=$1`
	return scan(r.db.QueryRow(ctx, q, key))
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status string) (payment.Payment, error) {
	q := `UPDATE payments SET status=$2, updated_at=now() WHERE id=$1
RETURNING id, order_id, amount, currency, description, status, payment_url, provider_payment_id, idempotency_key, created_at, updated_at`
	return scan(r.db.QueryRow(ctx, q, id, status))
}

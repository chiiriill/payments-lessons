package payments

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"stepik-payments-course/internal/common/apperror"
	"stepik-payments-course/internal/common/status"
	"stepik-payments-course/internal/usecase/checkPaymentUseCase"
	"stepik-payments-course/internal/usecase/createPaymentUseCase"
	"stepik-payments-course/internal/usecase/getPaymentUseCase"
	"stepik-payments-course/internal/usecase/receiveWebhookUseCase"
	"stepik-payments-course/internal/usecase/refundPaymentUseCase"
)

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{db: db}
}

func (r *Repo) CreatePaymentByIdempotencyKey(ctx context.Context, req createPaymentUseCase.Request) (createPaymentUseCase.Payment, bool, error) {
	if req.IdempotencyKey != "" {
		existing, err := r.getByIdempotencyKey(ctx, req.IdempotencyKey)
		if err == nil {
			return existing, true, nil
		}
		if !errors.Is(err, apperror.ErrNotFound) {
			return createPaymentUseCase.Payment{}, false, err
		}
	}

	paymentID, err := newID("pay")
	if err != nil {
		return createPaymentUseCase.Payment{}, false, err
	}
	q := `INSERT INTO payments (id, idempotency_key, order_id, amount, currency, description, status)
		VALUES ($1, NULLIF($2, ''), $3, $4, $5, $6, $7)
		RETURNING id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at`
	payment, err := scanCreatePayment(r.db.QueryRow(ctx, q, paymentID, req.IdempotencyKey, req.OrderID, req.Amount, req.Currency, req.Description, status.Pending))
	if err != nil {
		return createPaymentUseCase.Payment{}, false, err
	}
	return payment, false, nil
}

func (r *Repo) SetProviderDataForCreatePayment(ctx context.Context, paymentID string, providerPaymentID string, paymentURL string) (createPaymentUseCase.Payment, error) {
	q := `UPDATE payments SET provider_payment_id = $2, payment_url = $3, updated_at = now() WHERE id = $1
		RETURNING id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at`
	return scanCreatePayment(r.db.QueryRow(ctx, q, paymentID, providerPaymentID, paymentURL))
}

func (r *Repo) GetPaymentForGetPayment(ctx context.Context, paymentID string) (getPaymentUseCase.Payment, error) {
	q := `SELECT id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at FROM payments WHERE id = $1`
	return scanGetPayment(r.db.QueryRow(ctx, q, paymentID))
}

func (r *Repo) GetPaymentForCheckPayment(ctx context.Context, paymentID string) (checkPaymentUseCase.Payment, error) {
	q := `SELECT id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at FROM payments WHERE id = $1`
	return scanCheckPayment(r.db.QueryRow(ctx, q, paymentID))
}

func (r *Repo) SetPaidForCheckPayment(ctx context.Context, paymentID string) (checkPaymentUseCase.Payment, error) {
	q := `UPDATE payments SET status = $2, updated_at = now() WHERE id = $1
		RETURNING id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at`
	return scanCheckPayment(r.db.QueryRow(ctx, q, paymentID, status.Paid))
}

func (r *Repo) GetPaymentForRefundPayment(ctx context.Context, paymentID string) (refundPaymentUseCase.Payment, error) {
	q := `SELECT id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at FROM payments WHERE id = $1`
	return scanRefundPayment(r.db.QueryRow(ctx, q, paymentID))
}

func (r *Repo) SetRefundedForRefundPayment(ctx context.Context, paymentID string) (refundPaymentUseCase.Payment, error) {
	q := `UPDATE payments SET status = $2, updated_at = now() WHERE id = $1
		RETURNING id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at`
	return scanRefundPayment(r.db.QueryRow(ctx, q, paymentID, status.Refunded))
}

func (r *Repo) GetPaymentForReceiveWebhook(ctx context.Context, paymentID string) (receiveWebhookUseCase.Payment, error) {
	q := `SELECT id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at FROM payments WHERE id = $1`
	return scanReceiveWebhookPayment(r.db.QueryRow(ctx, q, paymentID))
}

func (r *Repo) SetPaidForReceiveWebhook(ctx context.Context, paymentID string) (receiveWebhookUseCase.Payment, error) {
	q := `UPDATE payments SET status = $2, updated_at = now() WHERE id = $1
		RETURNING id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at`
	return scanReceiveWebhookPayment(r.db.QueryRow(ctx, q, paymentID, status.Paid))
}

func (r *Repo) getByIdempotencyKey(ctx context.Context, key string) (createPaymentUseCase.Payment, error) {
	q := `SELECT id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at FROM payments WHERE idempotency_key = $1`
	return scanCreatePayment(r.db.QueryRow(ctx, q, key))
}

func newID(prefix string) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(b[:]), nil
}

func noRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.ErrNotFound
	}
	return err
}

func scanCreatePayment(row pgx.Row) (createPaymentUseCase.Payment, error) {
	var p createPaymentUseCase.Payment
	err := row.Scan(&p.ID, &p.OrderID, &p.Amount, &p.Currency, &p.Description, &p.Status, &p.ProviderPaymentID, &p.PaymentURL, &p.CreatedAt, &p.UpdatedAt)
	return p, noRows(err)
}

func scanGetPayment(row pgx.Row) (getPaymentUseCase.Payment, error) {
	var p getPaymentUseCase.Payment
	err := row.Scan(&p.ID, &p.OrderID, &p.Amount, &p.Currency, &p.Description, &p.Status, &p.ProviderPaymentID, &p.PaymentURL, &p.CreatedAt, &p.UpdatedAt)
	return p, noRows(err)
}

func scanCheckPayment(row pgx.Row) (checkPaymentUseCase.Payment, error) {
	var p checkPaymentUseCase.Payment
	err := row.Scan(&p.ID, &p.OrderID, &p.Amount, &p.Currency, &p.Description, &p.Status, &p.ProviderPaymentID, &p.PaymentURL, &p.CreatedAt, &p.UpdatedAt)
	return p, noRows(err)
}

func scanRefundPayment(row pgx.Row) (refundPaymentUseCase.Payment, error) {
	var p refundPaymentUseCase.Payment
	err := row.Scan(&p.ID, &p.OrderID, &p.Amount, &p.Currency, &p.Description, &p.Status, &p.ProviderPaymentID, &p.PaymentURL, &p.CreatedAt, &p.UpdatedAt)
	return p, noRows(err)
}

func scanReceiveWebhookPayment(row pgx.Row) (receiveWebhookUseCase.Payment, error) {
	var p receiveWebhookUseCase.Payment
	err := row.Scan(&p.ID, &p.OrderID, &p.Amount, &p.Currency, &p.Description, &p.Status, &p.ProviderPaymentID, &p.PaymentURL, &p.CreatedAt, &p.UpdatedAt)
	return p, noRows(err)
}

package payments

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"stepik-payments-course/internal/common/apperror"
	"stepik-payments-course/internal/common/status"
	"stepik-payments-course/internal/domain"
	"stepik-payments-course/internal/usecase/createPaymentUseCase"
)

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{db: db}
}

func (r *Repo) CreatePaymentByIdempotencyKey(ctx context.Context, req createPaymentUseCase.Request) (domain.Payment, bool, error) {
	if req.IdempotencyKey != "" {
		existing, err := r.getByIdempotencyKey(ctx, req.IdempotencyKey)
		if err == nil {
			return existing, true, nil
		}
		if !errors.Is(err, apperror.ErrNotFound) {
			return domain.Payment{}, false, err
		}
	}

	paymentID, err := newID("pay")
	if err != nil {
		return domain.Payment{}, false, err
	}
	q := `INSERT INTO payments (id, idempotency_key, order_id, amount, currency, description, status)
		VALUES ($1, NULLIF($2, ''), $3, $4, $5, $6, $7)
		RETURNING id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at`
	payment, err := scanPayment(r.db.QueryRow(ctx, q, paymentID, req.IdempotencyKey, req.OrderID, req.Amount, req.Currency, req.Description, status.Pending))
	if err != nil {
		if isUniqueViolation(err) && req.IdempotencyKey != "" {
			existing, getErr := r.getByIdempotencyKey(ctx, req.IdempotencyKey)
			if getErr == nil {
				return existing, true, nil
			}
		}
		return domain.Payment{}, false, err
	}
	return payment, false, nil
}

func (r *Repo) SetPaymentFailedForCreatePayment(ctx context.Context, paymentID string) error {
	q := `UPDATE payments SET status = $2, updated_at = now() WHERE id = $1`
	_, err := r.db.Exec(ctx, q, paymentID, status.Failed)
	return err
}

func (r *Repo) SetProviderDataForCreatePayment(ctx context.Context, paymentID string, providerPaymentID string, paymentURL string) (domain.Payment, error) {
	q := `UPDATE payments SET provider_payment_id = $2, payment_url = $3, updated_at = now() WHERE id = $1
		RETURNING id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at`
	return scanPayment(r.db.QueryRow(ctx, q, paymentID, providerPaymentID, paymentURL))
}

func (r *Repo) GetPaymentForGetPayment(ctx context.Context, paymentID string) (domain.Payment, error) {
	q := `SELECT id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at FROM payments WHERE id = $1`
	return scanPayment(r.db.QueryRow(ctx, q, paymentID))
}

func (r *Repo) GetPaymentForCheckPayment(ctx context.Context, paymentID string) (domain.Payment, error) {
	q := `SELECT id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at FROM payments WHERE id = $1`
	return scanPayment(r.db.QueryRow(ctx, q, paymentID))
}

func (r *Repo) SetPaidForCheckPayment(ctx context.Context, paymentID string) (domain.Payment, error) {
	q := `UPDATE payments SET status = $2, updated_at = now() WHERE id = $1 AND status = $3
		RETURNING id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at`
	p, err := scanPayment(r.db.QueryRow(ctx, q, paymentID, status.Paid, status.Pending))
	return p, noRowsAsConflict(err)
}

func (r *Repo) GetPaymentForRefundPayment(ctx context.Context, paymentID string) (domain.Payment, error) {
	q := `SELECT id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at FROM payments WHERE id = $1`
	return scanPayment(r.db.QueryRow(ctx, q, paymentID))
}

func (r *Repo) SetRefundedForRefundPayment(ctx context.Context, paymentID string) (domain.Payment, error) {
	q := `UPDATE payments SET status = $2, updated_at = now() WHERE id = $1 AND status = $3
		RETURNING id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at`
	p, err := scanPayment(r.db.QueryRow(ctx, q, paymentID, status.Refunded, status.Paid))
	return p, noRowsAsConflict(err)
}

func (r *Repo) MarkEventProcessedForReceiveWebhook(ctx context.Context, eventID string) (bool, error) {
	q := `INSERT INTO webhook_events (event_id) VALUES ($1) ON CONFLICT DO NOTHING`
	tag, err := r.db.Exec(ctx, q, eventID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repo) GetPaymentByProviderIDForReceiveWebhook(ctx context.Context, providerPaymentID string) (domain.Payment, error) {
	q := `SELECT id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at FROM payments WHERE provider_payment_id = $1`
	return scanPayment(r.db.QueryRow(ctx, q, providerPaymentID))
}

func (r *Repo) SetPaidForReceiveWebhook(ctx context.Context, paymentID string) (domain.Payment, error) {
	q := `UPDATE payments SET status = $2, updated_at = now() WHERE id = $1 AND status = $3
		RETURNING id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at`
	p, err := scanPayment(r.db.QueryRow(ctx, q, paymentID, status.Paid, status.Pending))
	return p, noRowsAsConflict(err)
}

func (r *Repo) SetPaidWithOutboxForReceiveWebhook(ctx context.Context, paymentID string) (domain.Payment, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.Payment{}, err
	}
	defer tx.Rollback(ctx)

	q := `UPDATE payments SET status = $2, updated_at = now() WHERE id = $1 AND status = $3
		RETURNING id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at`
	payment, err := scanPayment(tx.QueryRow(ctx, q, paymentID, status.Paid, status.Pending))
	if err != nil {
		return domain.Payment{}, noRowsAsConflict(err)
	}

	payload, _ := json.Marshal(map[string]any{
		"payment_id": payment.ID,
		"order_id":   payment.OrderID,
		"amount":     payment.Amount,
	})
	if _, err = tx.Exec(ctx, `INSERT INTO outbox_events (event_type, aggregate_id, payload) VALUES ($1, $2, $3)`, "payment.paid", paymentID, payload); err != nil {
		return domain.Payment{}, err
	}

	return payment, tx.Commit(ctx)
}

func (r *Repo) SetFailedForReceiveWebhook(ctx context.Context, paymentID string) (domain.Payment, error) {
	q := `UPDATE payments SET status = $2, updated_at = now() WHERE id = $1 AND status = $3
		RETURNING id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at`
	p, err := scanPayment(r.db.QueryRow(ctx, q, paymentID, status.Failed, status.Pending))
	return p, noRowsAsConflict(err)
}

func (r *Repo) GetStalePendingPaymentsForWorker(ctx context.Context, olderThan time.Duration) ([]domain.Payment, error) {
	threshold := time.Now().Add(-olderThan)
	q := `SELECT id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at
		FROM payments
		WHERE status = $1 AND provider_payment_id != '' AND updated_at < $2
		ORDER BY updated_at ASC
		LIMIT 100`
	rows, err := r.db.Query(ctx, q, status.Pending, threshold)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var payments []domain.Payment
	for rows.Next() {
		var p domain.Payment
		if err := rows.Scan(&p.ID, &p.OrderID, &p.Amount, &p.Currency, &p.Description, &p.Status, &p.ProviderPaymentID, &p.PaymentURL, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
}

func (r *Repo) SetPaidForWorker(ctx context.Context, paymentID string) (domain.Payment, error) {
	q := `UPDATE payments SET status = $2, updated_at = now() WHERE id = $1 AND status = $3
		RETURNING id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at`
	p, err := scanPayment(r.db.QueryRow(ctx, q, paymentID, status.Paid, status.Pending))
	return p, noRowsAsConflict(err)
}

func (r *Repo) getByIdempotencyKey(ctx context.Context, key string) (domain.Payment, error) {
	q := `SELECT id, order_id, amount, currency, description, status, provider_payment_id, payment_url, created_at, updated_at FROM payments WHERE idempotency_key = $1`
	return scanPayment(r.db.QueryRow(ctx, q, key))
}

func scanPayment(row pgx.Row) (domain.Payment, error) {
	var p domain.Payment
	err := row.Scan(&p.ID, &p.OrderID, &p.Amount, &p.Currency, &p.Description, &p.Status, &p.ProviderPaymentID, &p.PaymentURL, &p.CreatedAt, &p.UpdatedAt)
	return p, noRows(err)
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

func noRowsAsConflict(err error) error {
	if errors.Is(err, apperror.ErrNotFound) {
		return apperror.ErrInvalidTransition
	}
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

package payment

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	StatusPending  = "pending"
	StatusPaid     = "paid"
	StatusFailed   = "failed"
	StatusRefunded = "refunded"
)

var (
	ErrNotFound          = errors.New("payment not found")
	ErrInvalidTransition = errors.New("invalid payment status transition")
)

type Payment struct {
	ID                string    `json:"id" db:"id"`
	OrderID           string    `json:"order_id" db:"order_id"`
	Amount            int64     `json:"amount" db:"amount"`
	Currency          string    `json:"currency" db:"currency"`
	Description       string    `json:"description" db:"description"`
	Status            string    `json:"status" db:"status"`
	PaymentURL        string    `json:"payment_url" db:"payment_url"`
	ProviderPaymentID string    `json:"provider_payment_id" db:"provider_payment_id"`
	IdempotencyKey    *string   `json:"idempotency_key,omitempty" db:"idempotency_key"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

type CreateRequest struct {
	OrderID        string
	Amount         int64
	Currency       string
	Description    string
	IdempotencyKey *string
}

type Provider interface {
	CreatePayment(ctx context.Context, payment Payment) (providerID string, paymentURL string, err error)
	CheckPayment(ctx context.Context, providerPaymentID string) (status string, err error)
	RefundPayment(ctx context.Context, providerPaymentID string, amount int64) error
}

type Repository interface {
	Create(ctx context.Context, payment Payment) (Payment, error)
	Get(ctx context.Context, id string) (Payment, error)
	FindByIdempotencyKey(ctx context.Context, key string) (Payment, error)
	UpdateStatus(ctx context.Context, id string, status string) (Payment, error)
	UpdateProviderData(ctx context.Context, id string, providerID string, paymentURL string) (Payment, error)
}

type Service struct {
	repo     Repository
	provider Provider
}

func NewService(repo Repository, provider Provider) *Service {
	return &Service{repo: repo, provider: provider}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Payment, error) {
	if req.OrderID == "" || req.Amount <= 0 || req.Currency != "RUB" {
		return Payment{}, errors.New("order_id, positive amount and RUB currency are required")
	}
	if req.IdempotencyKey != nil && *req.IdempotencyKey != "" {
		existing, err := s.repo.FindByIdempotencyKey(ctx, *req.IdempotencyKey)
		if err == nil {
			return existing, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return Payment{}, err
		}
	}
	p := Payment{
		ID:             fmt.Sprintf("pay_%d", time.Now().UnixNano()),
		OrderID:        req.OrderID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Description:    req.Description,
		Status:         StatusPending,
		IdempotencyKey: req.IdempotencyKey,
	}
	created, err := s.repo.Create(ctx, p)
	if err != nil {
		return Payment{}, err
	}
	if s.provider == nil {
		return created, nil
	}
	providerID, paymentURL, err := s.provider.CreatePayment(ctx, created)
	if err != nil {
		return Payment{}, err
	}
	return s.repo.UpdateProviderData(ctx, created.ID, providerID, paymentURL)
}

func (s *Service) Get(ctx context.Context, id string) (Payment, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) MarkPaid(ctx context.Context, id string) (Payment, error) {
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return Payment{}, err
	}
	if p.Status != StatusPending {
		return Payment{}, ErrInvalidTransition
	}
	return s.repo.UpdateStatus(ctx, id, StatusPaid)
}

func (s *Service) Check(ctx context.Context, id string) (Payment, error) {
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return Payment{}, err
	}
	if s.provider == nil || p.ProviderPaymentID == "" {
		return p, nil
	}
	status, err := s.provider.CheckPayment(ctx, p.ProviderPaymentID)
	if err != nil {
		return Payment{}, err
	}
	if status == StatusPaid && p.Status == StatusPending {
		return s.repo.UpdateStatus(ctx, p.ID, StatusPaid)
	}
	if status == StatusFailed && p.Status == StatusPending {
		return s.repo.UpdateStatus(ctx, p.ID, StatusFailed)
	}
	return p, nil
}

func (s *Service) Refund(ctx context.Context, id string) (Payment, error) {
	p, err := s.repo.Get(ctx, id)
	if err != nil {
		return Payment{}, err
	}
	if p.Status != StatusPaid {
		return Payment{}, ErrInvalidTransition
	}
	if s.provider != nil && p.ProviderPaymentID != "" {
		if err := s.provider.RefundPayment(ctx, p.ProviderPaymentID, p.Amount); err != nil {
			return Payment{}, err
		}
	}
	return s.repo.UpdateStatus(ctx, id, StatusRefunded)
}

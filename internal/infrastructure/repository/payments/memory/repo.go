// Package memory provides an in-memory repository for use in tests.
// It implements the same contract as the Postgres repository.
package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"stepik-payments-course/internal/common/apperror"
	"stepik-payments-course/internal/common/status"
	"stepik-payments-course/internal/domain"
	"stepik-payments-course/internal/events"
	"stepik-payments-course/internal/usecase/createPaymentUseCase"
)

type Repo struct {
	mu          sync.Mutex
	payments    map[string]domain.Payment
	byIdem      map[string]string // idempotency_key → payment_id
	byProvider  map[string]string // provider_payment_id → payment_id
	seenEvents  map[string]bool   // webhook event_id deduplication
	outbox      []events.OutboxEvent
	outboxSeq   int64
}

func New() *Repo {
	return &Repo{
		payments:   make(map[string]domain.Payment),
		byIdem:     make(map[string]string),
		byProvider: make(map[string]string),
		seenEvents: make(map[string]bool),
	}
}

func (r *Repo) CreatePaymentByIdempotencyKey(_ context.Context, req createPaymentUseCase.Request) (domain.Payment, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if req.IdempotencyKey != "" {
		if id, ok := r.byIdem[req.IdempotencyKey]; ok {
			return r.payments[id], true, nil
		}
	}

	id, err := newID("pay")
	if err != nil {
		return domain.Payment{}, false, err
	}
	now := time.Now()
	p := domain.Payment{
		ID:          id,
		OrderID:     req.OrderID,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Description: req.Description,
		Status:      status.Pending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	r.payments[id] = p
	if req.IdempotencyKey != "" {
		r.byIdem[req.IdempotencyKey] = id
	}
	return p, false, nil
}

func (r *Repo) SetProviderDataForCreatePayment(_ context.Context, paymentID, providerPaymentID, paymentURL string) (domain.Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.payments[paymentID]
	if !ok {
		return domain.Payment{}, apperror.ErrNotFound
	}
	p.ProviderPaymentID = providerPaymentID
	p.PaymentURL = paymentURL
	p.UpdatedAt = time.Now()
	r.payments[paymentID] = p
	r.byProvider[providerPaymentID] = paymentID
	return p, nil
}

func (r *Repo) SetPaymentFailedForCreatePayment(_ context.Context, paymentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.payments[paymentID]
	if !ok {
		return apperror.ErrNotFound
	}
	p.Status = status.Failed
	p.UpdatedAt = time.Now()
	r.payments[paymentID] = p
	return nil
}

func (r *Repo) GetPaymentByProviderIDForReceiveWebhook(_ context.Context, providerPaymentID string) (domain.Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id, ok := r.byProvider[providerPaymentID]
	if !ok {
		return domain.Payment{}, apperror.ErrNotFound
	}
	return r.payments[id], nil
}

func (r *Repo) MarkEventProcessedForReceiveWebhook(_ context.Context, eventID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.seenEvents[eventID] {
		return false, nil
	}
	r.seenEvents[eventID] = true
	return true, nil
}

func (r *Repo) SetPaidWithOutboxForReceiveWebhook(_ context.Context, paymentID string) (domain.Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.payments[paymentID]
	if !ok {
		return domain.Payment{}, apperror.ErrNotFound
	}
	if !status.CanTransition(p.Status, status.Paid) {
		return domain.Payment{}, apperror.ErrInvalidTransition
	}
	p.Status = status.Paid
	p.UpdatedAt = time.Now()
	r.payments[paymentID] = p

	r.outboxSeq++
	r.outbox = append(r.outbox, events.OutboxEvent{
		ID:          r.outboxSeq,
		EventType:   "payment.paid",
		AggregateID: paymentID,
		Payload:     map[string]any{"payment_id": paymentID},
		CreatedAt:   time.Now(),
	})
	return p, nil
}

func (r *Repo) SetFailedForReceiveWebhook(_ context.Context, paymentID string) (domain.Payment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.payments[paymentID]
	if !ok {
		return domain.Payment{}, apperror.ErrNotFound
	}
	if !status.CanTransition(p.Status, status.Failed) {
		return domain.Payment{}, apperror.ErrInvalidTransition
	}
	p.Status = status.Failed
	p.UpdatedAt = time.Now()
	r.payments[paymentID] = p
	return p, nil
}

// OutboxEvents returns all outbox events written so far (for test assertions).
func (r *Repo) OutboxEvents() []events.OutboxEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]events.OutboxEvent, len(r.outbox))
	copy(cp, r.outbox)
	return cp
}

func newID(prefix string) (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(b[:]), nil
}

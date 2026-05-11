package payments_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"stepik-payments-course/internal/common/apperror"
	"stepik-payments-course/internal/common/status"
	payments "stepik-payments-course/internal/infrastructure/repository/payments"
	"stepik-payments-course/internal/usecase/createPaymentUseCase"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		fmt.Println("DATABASE_URL not set — skipping postgres integration tests")
		os.Exit(0)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect to test db: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	schema, err := os.ReadFile("testdata/schema.sql")
	if err != nil {
		fmt.Fprintf(os.Stderr, "read schema: %v\n", err)
		os.Exit(1)
	}
	if _, err := pool.Exec(ctx, string(schema)); err != nil {
		fmt.Fprintf(os.Stderr, "apply schema: %v\n", err)
		os.Exit(1)
	}

	testPool = pool
	os.Exit(m.Run())
}

func newRepo() *payments.Repo {
	return payments.NewRepo(testPool)
}

func truncate(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		_, err := testPool.Exec(context.Background(),
			"TRUNCATE payments, webhook_events, outbox_events RESTART IDENTITY CASCADE")
		if err != nil {
			t.Logf("truncate failed: %v", err)
		}
	})
}

// --- CreatePaymentByIdempotencyKey ---

func TestCreatePayment_NewPayment_HasPendingStatus(t *testing.T) {
	truncate(t)
	repo := newRepo()
	ctx := context.Background()

	p, existed, err := repo.CreatePaymentByIdempotencyKey(ctx, createPaymentUseCase.Request{
		OrderID:  "ord-1",
		Amount:   500,
		Currency: "RUB",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if existed {
		t.Error("expected existed=false for new payment")
	}
	if p.ID == "" {
		t.Error("expected non-empty payment id")
	}
	if p.Status != status.Pending {
		t.Errorf("expected status %q, got %q", status.Pending, p.Status)
	}
}

func TestCreatePayment_IdempotencyKey_ReturnsSamePayment(t *testing.T) {
	truncate(t)
	repo := newRepo()
	ctx := context.Background()

	req := createPaymentUseCase.Request{
		IdempotencyKey: "idem-key-1",
		OrderID:        "ord-2",
		Amount:         1000,
		Currency:       "RUB",
	}

	p1, existed1, err := repo.CreatePaymentByIdempotencyKey(ctx, req)
	if err != nil || existed1 {
		t.Fatalf("first call: err=%v existed=%v", err, existed1)
	}

	p2, existed2, err := repo.CreatePaymentByIdempotencyKey(ctx, req)
	if err != nil {
		t.Fatalf("second call: unexpected error: %v", err)
	}
	if !existed2 {
		t.Error("second call: expected existed=true")
	}
	if p1.ID != p2.ID {
		t.Errorf("idempotent calls must return same id: %q vs %q", p1.ID, p2.ID)
	}
}

// --- SetProviderDataForCreatePayment ---

func TestSetProviderData_UpdatesAndReturnsPayment(t *testing.T) {
	truncate(t)
	repo := newRepo()
	ctx := context.Background()

	p, _, _ := repo.CreatePaymentByIdempotencyKey(ctx, createPaymentUseCase.Request{
		OrderID: "ord-3", Amount: 200, Currency: "RUB",
	})

	updated, err := repo.SetProviderDataForCreatePayment(ctx, p.ID, "prov-100", "https://pay.example/prov-100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.ProviderPaymentID != "prov-100" {
		t.Errorf("unexpected provider id: %q", updated.ProviderPaymentID)
	}
	if updated.PaymentURL != "https://pay.example/prov-100" {
		t.Errorf("unexpected payment url: %q", updated.PaymentURL)
	}
}

// --- SetPaymentFailedForCreatePayment ---

func TestSetPaymentFailed_UpdatesStatus(t *testing.T) {
	truncate(t)
	repo := newRepo()
	ctx := context.Background()

	p, _, _ := repo.CreatePaymentByIdempotencyKey(ctx, createPaymentUseCase.Request{
		OrderID: "ord-4", Amount: 300, Currency: "RUB",
	})

	if err := repo.SetPaymentFailedForCreatePayment(ctx, p.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify: setting paid on a failed payment must fail the transition
	_, err := repo.SetPaidWithOutboxForReceiveWebhook(ctx, p.ID)
	if !errors.Is(err, apperror.ErrInvalidTransition) {
		t.Errorf("expected ErrInvalidTransition for failed→paid, got %v", err)
	}
}

// --- MarkEventProcessedForReceiveWebhook ---

func TestMarkEvent_Deduplication(t *testing.T) {
	truncate(t)
	repo := newRepo()
	ctx := context.Background()

	isNew1, err := repo.MarkEventProcessedForReceiveWebhook(ctx, "evt-xyz")
	if err != nil || !isNew1 {
		t.Fatalf("first call: err=%v isNew=%v", err, isNew1)
	}

	isNew2, err := repo.MarkEventProcessedForReceiveWebhook(ctx, "evt-xyz")
	if err != nil {
		t.Fatalf("second call: unexpected error: %v", err)
	}
	if isNew2 {
		t.Error("second call: expected isNew=false for duplicate event id")
	}
}

// --- SetPaidWithOutboxForReceiveWebhook ---

func TestSetPaidWithOutbox_StatusAndOutboxWrittenTogether(t *testing.T) {
	truncate(t)
	repo := newRepo()
	ctx := context.Background()

	p, _, _ := repo.CreatePaymentByIdempotencyKey(ctx, createPaymentUseCase.Request{
		OrderID: "ord-5", Amount: 400, Currency: "RUB",
	})
	repo.SetProviderDataForCreatePayment(ctx, p.ID, "prov-200", "")

	paid, err := repo.SetPaidWithOutboxForReceiveWebhook(ctx, p.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if paid.Status != status.Paid {
		t.Errorf("expected status paid, got %q", paid.Status)
	}

	events, err := repo.GetPendingOutboxEventsForWorker(ctx, 10)
	if err != nil {
		t.Fatalf("GetPendingOutboxEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(events))
	}
	if events[0].EventType != "payment.paid" {
		t.Errorf("unexpected event type: %q", events[0].EventType)
	}
	if events[0].AggregateID != p.ID {
		t.Errorf("unexpected aggregate id: %q", events[0].AggregateID)
	}
}

func TestSetPaidWithOutbox_InvalidTransition_ReturnsError(t *testing.T) {
	truncate(t)
	repo := newRepo()
	ctx := context.Background()

	p, _, _ := repo.CreatePaymentByIdempotencyKey(ctx, createPaymentUseCase.Request{
		OrderID: "ord-6", Amount: 500, Currency: "RUB",
	})
	repo.SetPaymentFailedForCreatePayment(ctx, p.ID)

	_, err := repo.SetPaidWithOutboxForReceiveWebhook(ctx, p.ID)
	if !errors.Is(err, apperror.ErrInvalidTransition) {
		t.Errorf("expected ErrInvalidTransition, got %v", err)
	}
}

// --- GetPendingOutboxEventsForWorker / MarkOutboxEventPublishedForWorker ---

func TestOutboxWorker_PublishCycle(t *testing.T) {
	truncate(t)
	repo := newRepo()
	ctx := context.Background()

	// Create payment and trigger paid transition (writes outbox event)
	p, _, _ := repo.CreatePaymentByIdempotencyKey(ctx, createPaymentUseCase.Request{
		OrderID: "ord-7", Amount: 600, Currency: "RUB",
	})
	repo.SetProviderDataForCreatePayment(ctx, p.ID, "prov-300", "")
	repo.SetPaidWithOutboxForReceiveWebhook(ctx, p.ID)

	pending, err := repo.GetPendingOutboxEventsForWorker(ctx, 10)
	if err != nil || len(pending) != 1 {
		t.Fatalf("expected 1 pending event, got %d (err: %v)", len(pending), err)
	}
	eventID := pending[0].ID

	if err := repo.MarkOutboxEventPublishedForWorker(ctx, eventID); err != nil {
		t.Fatalf("mark published: %v", err)
	}

	after, err := repo.GetPendingOutboxEventsForWorker(ctx, 10)
	if err != nil {
		t.Fatalf("second GetPending: %v", err)
	}
	if len(after) != 0 {
		t.Errorf("expected 0 pending events after publish, got %d", len(after))
	}
}

// --- GetPaymentByProviderIDForReceiveWebhook ---

func TestGetByProviderID_NotFound(t *testing.T) {
	truncate(t)
	repo := newRepo()

	_, err := repo.GetPaymentByProviderIDForReceiveWebhook(context.Background(), "prov-nonexistent")
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetByProviderID_FoundAfterUpdate(t *testing.T) {
	truncate(t)
	repo := newRepo()
	ctx := context.Background()

	p, _, _ := repo.CreatePaymentByIdempotencyKey(ctx, createPaymentUseCase.Request{
		OrderID: "ord-8", Amount: 700, Currency: "RUB",
	})
	repo.SetProviderDataForCreatePayment(ctx, p.ID, "prov-lookup", "")

	found, err := repo.GetPaymentByProviderIDForReceiveWebhook(ctx, "prov-lookup")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.ID != p.ID {
		t.Errorf("expected payment id %q, got %q", p.ID, found.ID)
	}
}

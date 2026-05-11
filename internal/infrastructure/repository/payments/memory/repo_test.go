package memory

import (
	"context"
	"errors"
	"testing"

	"stepik-payments-course/internal/common/apperror"
	"stepik-payments-course/internal/common/status"
	"stepik-payments-course/internal/usecase/createPaymentUseCase"
)

// --- CreatePaymentByIdempotencyKey ---

func TestCreatePayment_NoKey_AlwaysCreatesNew(t *testing.T) {
	r := New()
	ctx := context.Background()

	req := createPaymentUseCase.Request{OrderID: "ord-1", Amount: 1000, Currency: "RUB"}

	p1, existed1, err := r.CreatePaymentByIdempotencyKey(ctx, req)
	if err != nil || existed1 {
		t.Fatalf("expected new payment, got err=%v existed=%v", err, existed1)
	}

	p2, existed2, err := r.CreatePaymentByIdempotencyKey(ctx, req)
	if err != nil || existed2 {
		t.Fatalf("expected new payment on second call, got err=%v existed=%v", err, existed2)
	}

	if p1.ID == p2.ID {
		t.Error("expected distinct IDs for requests without idempotency key")
	}
}

func TestCreatePayment_WithKey_IdempotentOnRepeat(t *testing.T) {
	r := New()
	ctx := context.Background()

	req := createPaymentUseCase.Request{IdempotencyKey: "key-abc", OrderID: "ord-2"}

	p1, existed1, err := r.CreatePaymentByIdempotencyKey(ctx, req)
	if err != nil || existed1 {
		t.Fatalf("first call: expected new payment, got err=%v existed=%v", err, existed1)
	}

	p2, existed2, err := r.CreatePaymentByIdempotencyKey(ctx, req)
	if err != nil {
		t.Fatalf("second call: unexpected error: %v", err)
	}
	if !existed2 {
		t.Error("second call: expected existed=true")
	}
	if p1.ID != p2.ID {
		t.Errorf("idempotent calls must return same payment: %q vs %q", p1.ID, p2.ID)
	}
}

func TestCreatePayment_InitialStatus_IsPending(t *testing.T) {
	r := New()
	p, _, err := r.CreatePaymentByIdempotencyKey(context.Background(), createPaymentUseCase.Request{})
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != status.Pending {
		t.Errorf("expected status %q, got %q", status.Pending, p.Status)
	}
}

// --- SetProviderDataForCreatePayment ---

func TestSetProviderData_UpdatesFields(t *testing.T) {
	r := New()
	ctx := context.Background()

	p, _, _ := r.CreatePaymentByIdempotencyKey(ctx, createPaymentUseCase.Request{})

	updated, err := r.SetProviderDataForCreatePayment(ctx, p.ID, "prov-1", "https://pay.example/prov-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.ProviderPaymentID != "prov-1" {
		t.Errorf("expected provider id %q, got %q", "prov-1", updated.ProviderPaymentID)
	}
	if updated.PaymentURL != "https://pay.example/prov-1" {
		t.Errorf("unexpected payment url: %q", updated.PaymentURL)
	}
}

func TestSetProviderData_NotFound(t *testing.T) {
	r := New()
	_, err := r.SetProviderDataForCreatePayment(context.Background(), "pay_nonexistent", "prov-1", "")
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// --- SetPaymentFailedForCreatePayment ---

func TestSetPaymentFailed_MarksAsFailed(t *testing.T) {
	r := New()
	ctx := context.Background()

	p, _, _ := r.CreatePaymentByIdempotencyKey(ctx, createPaymentUseCase.Request{})

	if err := r.SetPaymentFailedForCreatePayment(ctx, p.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify by looking up via provider path (or direct state check via SetProviderData returning fresh state)
	_, err := r.SetPaidWithOutboxForReceiveWebhook(ctx, p.ID)
	if !errors.Is(err, apperror.ErrInvalidTransition) {
		t.Errorf("failed payment must not transition to paid; got err=%v", err)
	}
}

// --- MarkEventProcessedForReceiveWebhook ---

func TestMarkEvent_FirstCall_ReturnsTrue(t *testing.T) {
	r := New()
	isNew, err := r.MarkEventProcessedForReceiveWebhook(context.Background(), "evt-1")
	if err != nil {
		t.Fatal(err)
	}
	if !isNew {
		t.Error("expected isNew=true for first occurrence")
	}
}

func TestMarkEvent_SecondCall_ReturnsFalse(t *testing.T) {
	r := New()
	ctx := context.Background()

	r.MarkEventProcessedForReceiveWebhook(ctx, "evt-dup")
	isNew, err := r.MarkEventProcessedForReceiveWebhook(ctx, "evt-dup")
	if err != nil {
		t.Fatal(err)
	}
	if isNew {
		t.Error("expected isNew=false for duplicate event")
	}
}

// --- SetPaidWithOutboxForReceiveWebhook ---

func TestSetPaidWithOutbox_TransitionsPending_WritesOutbox(t *testing.T) {
	r := New()
	ctx := context.Background()

	p, _, _ := r.CreatePaymentByIdempotencyKey(ctx, createPaymentUseCase.Request{})
	r.SetProviderDataForCreatePayment(ctx, p.ID, "prov-1", "")
	r.GetPaymentByProviderIDForReceiveWebhook(ctx, "prov-1") // warm up byProvider index

	paid, err := r.SetPaidWithOutboxForReceiveWebhook(ctx, p.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if paid.Status != status.Paid {
		t.Errorf("expected status paid, got %q", paid.Status)
	}

	outbox := r.OutboxEvents()
	if len(outbox) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(outbox))
	}
	if outbox[0].EventType != "payment.paid" {
		t.Errorf("unexpected outbox event type: %q", outbox[0].EventType)
	}
	if outbox[0].AggregateID != p.ID {
		t.Errorf("unexpected outbox aggregate id: %q", outbox[0].AggregateID)
	}
}

func TestSetPaidWithOutbox_InvalidTransition_ReturnsError(t *testing.T) {
	r := New()
	ctx := context.Background()

	p, _, _ := r.CreatePaymentByIdempotencyKey(ctx, createPaymentUseCase.Request{})
	r.SetPaymentFailedForCreatePayment(ctx, p.ID) // now status=failed

	_, err := r.SetPaidWithOutboxForReceiveWebhook(ctx, p.ID)
	if !errors.Is(err, apperror.ErrInvalidTransition) {
		t.Errorf("expected ErrInvalidTransition, got %v", err)
	}
}

// --- GetPaymentByProviderIDForReceiveWebhook ---

func TestGetByProviderID_NotFound(t *testing.T) {
	r := New()
	_, err := r.GetPaymentByProviderIDForReceiveWebhook(context.Background(), "prov-unknown")
	if !errors.Is(err, apperror.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetByProviderID_FoundAfterSetProviderData(t *testing.T) {
	r := New()
	ctx := context.Background()

	p, _, _ := r.CreatePaymentByIdempotencyKey(ctx, createPaymentUseCase.Request{})
	r.SetProviderDataForCreatePayment(ctx, p.ID, "prov-99", "")

	found, err := r.GetPaymentByProviderIDForReceiveWebhook(ctx, "prov-99")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.ID != p.ID {
		t.Errorf("expected payment id %q, got %q", p.ID, found.ID)
	}
}

package receiveWebhookUseCase

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"stepik-payments-course/internal/common/status"
	"stepik-payments-course/internal/domain"
)

// --- mocks ---

type mockRepo struct {
	markEventFn  func(ctx context.Context, eventID string) (bool, error)
	getPaymentFn func(ctx context.Context, providerPaymentID string) (domain.Payment, error)
	setPaidFn    func(ctx context.Context, paymentID string) (domain.Payment, error)
	setFailedFn  func(ctx context.Context, paymentID string) (domain.Payment, error)
}

func (m *mockRepo) MarkEventProcessedForReceiveWebhook(ctx context.Context, eventID string) (bool, error) {
	return m.markEventFn(ctx, eventID)
}

func (m *mockRepo) GetPaymentByProviderIDForReceiveWebhook(ctx context.Context, providerPaymentID string) (domain.Payment, error) {
	return m.getPaymentFn(ctx, providerPaymentID)
}

func (m *mockRepo) SetPaidWithOutboxForReceiveWebhook(ctx context.Context, paymentID string) (domain.Payment, error) {
	if m.setPaidFn == nil {
		return domain.Payment{}, nil
	}
	return m.setPaidFn(ctx, paymentID)
}

func (m *mockRepo) SetFailedForReceiveWebhook(ctx context.Context, paymentID string) (domain.Payment, error) {
	if m.setFailedFn == nil {
		return domain.Payment{}, nil
	}
	return m.setFailedFn(ctx, paymentID)
}

func nopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// --- tests ---

func TestExecute_PaymentSucceeded_SetsPaid(t *testing.T) {
	payment := domain.Payment{ID: "pay-1", Status: status.Pending}
	paidCalled := false

	repo := &mockRepo{
		markEventFn:  func(_ context.Context, _ string) (bool, error) { return true, nil },
		getPaymentFn: func(_ context.Context, _ string) (domain.Payment, error) { return payment, nil },
		setPaidFn:    func(_ context.Context, _ string) (domain.Payment, error) { paidCalled = true; return domain.Payment{}, nil },
	}

	uc := New(repo, nopLogger())
	err := uc.Execute(context.Background(), Request{
		EventID:           "evt-1",
		ProviderPaymentID: "prov-1",
		Event:             "payment.succeeded",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !paidCalled {
		t.Error("expected SetPaidWithOutbox to be called")
	}
}

func TestExecute_PaymentFailed_SetsFailed(t *testing.T) {
	payment := domain.Payment{ID: "pay-1", Status: status.Pending}
	failedCalled := false

	repo := &mockRepo{
		markEventFn:  func(_ context.Context, _ string) (bool, error) { return true, nil },
		getPaymentFn: func(_ context.Context, _ string) (domain.Payment, error) { return payment, nil },
		setFailedFn:  func(_ context.Context, _ string) (domain.Payment, error) { failedCalled = true; return domain.Payment{}, nil },
	}

	uc := New(repo, nopLogger())
	err := uc.Execute(context.Background(), Request{
		EventID:           "evt-2",
		ProviderPaymentID: "prov-1",
		Event:             "payment.failed",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !failedCalled {
		t.Error("expected SetFailed to be called")
	}
}

func TestExecute_UnknownEvent_NoRepoCall(t *testing.T) {
	repo := &mockRepo{
		markEventFn: func(_ context.Context, _ string) (bool, error) {
			t.Fatal("repo must not be called for unknown event")
			return false, nil
		},
	}

	uc := New(repo, nopLogger())
	err := uc.Execute(context.Background(), Request{Event: "payment.pending"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecute_DuplicateEvent_Ignored(t *testing.T) {
	getPaymentCalled := false

	repo := &mockRepo{
		markEventFn:  func(_ context.Context, _ string) (bool, error) { return false, nil },
		getPaymentFn: func(_ context.Context, _ string) (domain.Payment, error) { getPaymentCalled = true; return domain.Payment{}, nil },
	}

	uc := New(repo, nopLogger())
	err := uc.Execute(context.Background(), Request{
		EventID: "evt-dup",
		Event:   "payment.succeeded",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if getPaymentCalled {
		t.Error("payment lookup must be skipped for duplicate event")
	}
}

func TestExecute_InvalidTransition_NoUpdate(t *testing.T) {
	payment := domain.Payment{ID: "pay-1", Status: status.Failed}
	setPaidCalled := false

	repo := &mockRepo{
		markEventFn:  func(_ context.Context, _ string) (bool, error) { return true, nil },
		getPaymentFn: func(_ context.Context, _ string) (domain.Payment, error) { return payment, nil },
		setPaidFn:    func(_ context.Context, _ string) (domain.Payment, error) { setPaidCalled = true; return domain.Payment{}, nil },
	}

	uc := New(repo, nopLogger())
	err := uc.Execute(context.Background(), Request{
		EventID: "evt-3",
		Event:   "payment.succeeded",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if setPaidCalled {
		t.Error("SetPaid must not be called when transition is invalid")
	}
}

func TestExecute_MarkEventError_ReturnsError(t *testing.T) {
	dbErr := errors.New("db error")
	repo := &mockRepo{
		markEventFn: func(_ context.Context, _ string) (bool, error) { return false, dbErr },
	}

	uc := New(repo, nopLogger())
	err := uc.Execute(context.Background(), Request{
		EventID: "evt-4",
		Event:   "payment.succeeded",
	})
	if !errors.Is(err, dbErr) {
		t.Fatalf("expected db error, got %v", err)
	}
}

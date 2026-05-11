package createPaymentUseCase

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
	createFn     func(ctx context.Context, req Request) (domain.Payment, bool, error)
	setDataFn    func(ctx context.Context, paymentID, providerPaymentID, paymentURL string) (domain.Payment, error)
	setFailedFn  func(ctx context.Context, paymentID string) error
}

func (m *mockRepo) CreatePaymentByIdempotencyKey(ctx context.Context, req Request) (domain.Payment, bool, error) {
	return m.createFn(ctx, req)
}

func (m *mockRepo) SetProviderDataForCreatePayment(ctx context.Context, paymentID, providerPaymentID, paymentURL string) (domain.Payment, error) {
	return m.setDataFn(ctx, paymentID, providerPaymentID, paymentURL)
}

func (m *mockRepo) SetPaymentFailedForCreatePayment(ctx context.Context, paymentID string) error {
	return m.setFailedFn(ctx, paymentID)
}

type mockProvider struct {
	createFn func(ctx context.Context, req ProviderCreateRequest) (ProviderCreateResponse, error)
}

func (m *mockProvider) CreatePayment(ctx context.Context, req ProviderCreateRequest) (ProviderCreateResponse, error) {
	return m.createFn(ctx, req)
}

func nopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// --- tests ---

func TestExecute_NewPayment_Success(t *testing.T) {
	payment := domain.Payment{ID: "pay-1", OrderID: "ord-1", Status: status.Pending}
	final := domain.Payment{ID: "pay-1", ProviderPaymentID: "prov-1", PaymentURL: "https://pay.example"}

	repo := &mockRepo{
		createFn:  func(_ context.Context, _ Request) (domain.Payment, bool, error) { return payment, false, nil },
		setDataFn: func(_ context.Context, _, _, _ string) (domain.Payment, error) { return final, nil },
	}
	prov := &mockProvider{
		createFn: func(_ context.Context, _ ProviderCreateRequest) (ProviderCreateResponse, error) {
			return ProviderCreateResponse{ProviderPaymentID: "prov-1", PaymentURL: "https://pay.example"}, nil
		},
	}

	uc := New(repo, prov, nopLogger())
	got, err := uc.Execute(context.Background(), Request{IdempotencyKey: "key-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ProviderPaymentID != "prov-1" {
		t.Errorf("expected provider payment id %q, got %q", "prov-1", got.ProviderPaymentID)
	}
}

func TestExecute_Idempotent_AlreadyHasProviderData(t *testing.T) {
	existing := domain.Payment{ID: "pay-1", ProviderPaymentID: "prov-1", Status: status.Pending}

	repo := &mockRepo{
		createFn: func(_ context.Context, _ Request) (domain.Payment, bool, error) { return existing, true, nil },
	}
	prov := &mockProvider{
		createFn: func(_ context.Context, _ ProviderCreateRequest) (ProviderCreateResponse, error) {
			t.Fatal("provider must not be called for existing payment with provider data")
			return ProviderCreateResponse{}, nil
		},
	}

	uc := New(repo, prov, nopLogger())
	got, err := uc.Execute(context.Background(), Request{IdempotencyKey: "key-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "pay-1" {
		t.Errorf("expected existing payment id, got %q", got.ID)
	}
}

func TestExecute_Idempotent_RetryAfterPartialFailure(t *testing.T) {
	existing := domain.Payment{ID: "pay-1", Status: status.Pending}
	final := domain.Payment{ID: "pay-1", ProviderPaymentID: "prov-2"}

	provCalled := 0
	repo := &mockRepo{
		createFn:  func(_ context.Context, _ Request) (domain.Payment, bool, error) { return existing, true, nil },
		setDataFn: func(_ context.Context, _, _, _ string) (domain.Payment, error) { return final, nil },
	}
	prov := &mockProvider{
		createFn: func(_ context.Context, _ ProviderCreateRequest) (ProviderCreateResponse, error) {
			provCalled++
			return ProviderCreateResponse{ProviderPaymentID: "prov-2"}, nil
		},
	}

	uc := New(repo, prov, nopLogger())
	if _, err := uc.Execute(context.Background(), Request{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provCalled != 1 {
		t.Errorf("expected provider called once, got %d", provCalled)
	}
}

func TestExecute_ProviderError_MarksPaymentFailed(t *testing.T) {
	payment := domain.Payment{ID: "pay-1", Status: status.Pending}
	provErr := errors.New("provider unavailable")
	failedID := ""

	repo := &mockRepo{
		createFn:    func(_ context.Context, _ Request) (domain.Payment, bool, error) { return payment, false, nil },
		setFailedFn: func(_ context.Context, id string) error { failedID = id; return nil },
	}
	prov := &mockProvider{
		createFn: func(_ context.Context, _ ProviderCreateRequest) (ProviderCreateResponse, error) {
			return ProviderCreateResponse{}, provErr
		},
	}

	uc := New(repo, prov, nopLogger())
	_, err := uc.Execute(context.Background(), Request{})
	if !errors.Is(err, provErr) {
		t.Fatalf("expected provider error, got %v", err)
	}
	if failedID != "pay-1" {
		t.Errorf("expected SetPaymentFailed called with pay-1, got %q", failedID)
	}
}

package mockclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"stepik-payments-course/internal/usecase/createPaymentUseCase"
)

func newClient(baseURL string) *Client {
	return New(baseURL, 5*time.Second, 0)
}

func newClientWithRetries(baseURL string, retries int) *Client {
	return New(baseURL, 5*time.Second, retries)
}

// --- CreatePayment ---

func TestCreatePayment_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/payments" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		json.NewEncoder(w).Encode(createPaymentUseCase.ProviderCreateResponse{
			ProviderPaymentID: "prov-42",
			PaymentURL:        "https://pay.example/prov-42",
		})
	}))
	defer srv.Close()

	res, err := newClient(srv.URL).CreatePayment(context.Background(), createPaymentUseCase.ProviderCreateRequest{
		PaymentID: "pay-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ProviderPaymentID != "prov-42" {
		t.Errorf("unexpected provider id: %q", res.ProviderPaymentID)
	}
	if res.PaymentURL != "https://pay.example/prov-42" {
		t.Errorf("unexpected payment url: %q", res.PaymentURL)
	}
}

func TestCreatePayment_ServerError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := newClient(srv.URL).CreatePayment(context.Background(), createPaymentUseCase.ProviderCreateRequest{})
	if err == nil {
		t.Fatal("expected error for 5xx response, got nil")
	}
}

func TestCreatePayment_ClientError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	_, err := newClient(srv.URL).CreatePayment(context.Background(), createPaymentUseCase.ProviderCreateRequest{})
	if err == nil {
		t.Fatal("expected error for 4xx response, got nil")
	}
}

// --- Retry logic ---

func TestRetry_RetriesOnServerError(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(createPaymentUseCase.ProviderCreateResponse{ProviderPaymentID: "prov-ok"})
	}))
	defer srv.Close()

	res, err := newClientWithRetries(srv.URL, 2).CreatePayment(context.Background(), createPaymentUseCase.ProviderCreateRequest{})
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if res.ProviderPaymentID != "prov-ok" {
		t.Errorf("unexpected provider id: %q", res.ProviderPaymentID)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetry_NoRetryOnClientError(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer srv.Close()

	_, err := newClientWithRetries(srv.URL, 3).CreatePayment(context.Background(), createPaymentUseCase.ProviderCreateRequest{})
	if err == nil {
		t.Fatal("expected error for 4xx, got nil")
	}
	if attempts != 1 {
		t.Errorf("expected exactly 1 attempt for 4xx, got %d", attempts)
	}
}

// --- Context cancellation ---

func TestCreatePayment_CancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := newClientWithRetries(srv.URL, 5).CreatePayment(ctx, createPaymentUseCase.ProviderCreateRequest{})
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}

package handler_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"stepik-payments-course/internal/domain"
	handler "stepik-payments-course/internal/infrastructure/http/handler"
	mockprovider "stepik-payments-course/internal/infrastructure/provider/mock"
	"stepik-payments-course/internal/infrastructure/repository/payments/memory"
	"stepik-payments-course/internal/usecase/createPaymentUseCase"
	"stepik-payments-course/internal/usecase/receiveWebhookUseCase"
)

const testSecret = "test-webhook-secret"

// newTestApp wires a Fiber app with in-memory repo and mock provider.
// Only createPayment and receiveWebhook use cases are wired; other handlers are nil
// and must not be called in tests.
func newTestApp(t *testing.T) (*fiber.App, *memory.Repo) {
	t.Helper()
	repo := memory.New()
	prov := mockprovider.NewProvider()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	h := handler.New(
		createPaymentUseCase.New(repo, prov, logger),
		nil, nil, nil,
		receiveWebhookUseCase.New(repo, logger),
		testSecret,
		func(_ context.Context) error { return nil },
	)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	h.Register(app)
	return app, repo
}

func signBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func jsonBody(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(b)
}

// --- POST /payments ---

func TestCreatePayment_ValidRequest_Returns201(t *testing.T) {
	app, _ := newTestApp(t)

	body, _ := json.Marshal(map[string]any{
		"order_id": "ord-1", "amount": 500, "currency": "RUB",
	})
	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	var p domain.Payment
	json.NewDecoder(resp.Body).Decode(&p)
	if p.ID == "" {
		t.Error("expected non-empty payment id in response")
	}
}

func TestCreatePayment_MissingOrderID_Returns400(t *testing.T) {
	app, _ := newTestApp(t)

	body, _ := json.Marshal(map[string]any{"amount": 500, "currency": "RUB"})
	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreatePayment_UnsupportedCurrency_Returns400(t *testing.T) {
	app, _ := newTestApp(t)

	body, _ := json.Marshal(map[string]any{
		"order_id": "ord-2", "amount": 500, "currency": "USD",
	})
	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestCreatePayment_IdempotencyKey_ReturnsSamePaymentID(t *testing.T) {
	app, _ := newTestApp(t)

	body, _ := json.Marshal(map[string]any{
		"order_id": "ord-3", "amount": 1000, "currency": "RUB",
	})

	makeReq := func() *domain.Payment {
		req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "idem-key-abc")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.StatusCode)
		}
		var p domain.Payment
		json.NewDecoder(resp.Body).Decode(&p)
		return &p
	}

	p1 := makeReq()
	p2 := makeReq()

	if p1.ID != p2.ID {
		t.Errorf("idempotency key must return same payment id: %q vs %q", p1.ID, p2.ID)
	}
}

// --- POST /webhooks/mock-provider ---

func TestWebhook_ValidSignature_Returns200(t *testing.T) {
	app, repo := newTestApp(t)
	ctx := context.Background()

	// Seed: create payment with provider data
	p, _, _ := repo.CreatePaymentByIdempotencyKey(ctx, createPaymentUseCase.Request{
		OrderID: "ord-w1", Amount: 200, Currency: "RUB",
	})
	repo.SetProviderDataForCreatePayment(ctx, p.ID, "prov-w1", "")

	payload, _ := json.Marshal(map[string]any{
		"event_id":            "evt-1",
		"provider_payment_id": "prov-w1",
		"event":               "payment.succeeded",
	})
	req := httptest.NewRequest(http.MethodPost, "/webhooks/mock-provider", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Provider-Signature", signBody(testSecret, payload))

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected 200, got %d: %s", resp.StatusCode, body)
	}
}

func TestWebhook_InvalidSignature_Returns401(t *testing.T) {
	app, _ := newTestApp(t)

	payload, _ := json.Marshal(map[string]any{
		"event_id": "evt-2", "provider_payment_id": "prov-w2", "event": "payment.succeeded",
	})
	req := httptest.NewRequest(http.MethodPost, "/webhooks/mock-provider", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Provider-Signature", "deadbeef")

	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestWebhook_DuplicateEventID_Returns200(t *testing.T) {
	app, repo := newTestApp(t)
	ctx := context.Background()

	p, _, _ := repo.CreatePaymentByIdempotencyKey(ctx, createPaymentUseCase.Request{
		OrderID: "ord-w3", Amount: 300, Currency: "RUB",
	})
	repo.SetProviderDataForCreatePayment(ctx, p.ID, "prov-w3", "")

	payload, _ := json.Marshal(map[string]any{
		"event_id":            "evt-dup",
		"provider_payment_id": "prov-w3",
		"event":               "payment.succeeded",
	})
	sig := signBody(testSecret, payload)

	send := func() int {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/mock-provider", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Provider-Signature", sig)
		resp, _ := app.Test(req)
		return resp.StatusCode
	}

	if code := send(); code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", code)
	}
	if code := send(); code != http.StatusOK {
		t.Errorf("duplicate request: expected 200 (idempotent), got %d", code)
	}
}

func TestWebhook_UnknownEventType_Returns200(t *testing.T) {
	app, _ := newTestApp(t)

	payload, _ := json.Marshal(map[string]any{
		"event_id":            "evt-unk",
		"provider_payment_id": "prov-unk",
		"event":               "payment.pending",
	})
	req := httptest.NewRequest(http.MethodPost, "/webhooks/mock-provider", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Provider-Signature", signBody(testSecret, payload))

	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for unknown event (should be ignored), got %d", resp.StatusCode)
	}
}

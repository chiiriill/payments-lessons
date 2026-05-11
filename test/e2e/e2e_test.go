package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
)

var baseURL string

func TestMain(m *testing.M) {
	baseURL = os.Getenv("E2E_BASE_URL")
	if baseURL == "" {
		fmt.Println("E2E_BASE_URL not set, skipping e2e tests")
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestHealthCheck(t *testing.T) {
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestCreateAndGetPayment(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"order_id":    "e2e-order-1",
		"amount":      1000,
		"currency":    "RUB",
		"description": "e2e test payment",
	})

	resp, err := http.Post(baseURL+"/payments", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create payment: expected 201, got %d", resp.StatusCode)
	}

	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}

	id, ok := created["id"].(string)
	if !ok || id == "" {
		t.Fatal("expected non-empty id in response")
	}
	t.Logf("created payment id=%s", id)

	get, err := http.Get(baseURL + "/payments/" + id)
	if err != nil {
		t.Fatal(err)
	}
	defer get.Body.Close()

	if get.StatusCode != http.StatusOK {
		t.Fatalf("get payment: expected 200, got %d", get.StatusCode)
	}

	var fetched map[string]any
	if err := json.NewDecoder(get.Body).Decode(&fetched); err != nil {
		t.Fatal(err)
	}

	if fetched["id"] != id {
		t.Fatalf("expected id=%s, got %v", id, fetched["id"])
	}
	if fetched["status"] != "pending" {
		t.Fatalf("expected status=pending, got %v", fetched["status"])
	}
}

func TestIdempotency(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"order_id": "e2e-idem-order",
		"amount":   500,
		"currency": "RUB",
	})

	makeRequest := func() string {
		req, _ := http.NewRequest(http.MethodPost, baseURL+"/payments", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", "e2e-idem-key-1")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201, got %d", resp.StatusCode)
		}
		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)
		return result["id"].(string)
	}

	id1 := makeRequest()
	id2 := makeRequest()

	if id1 != id2 {
		t.Fatalf("idempotency broken: first=%s second=%s", id1, id2)
	}
	t.Logf("idempotency ok: both calls returned id=%s", id1)
}

func TestCreatePaymentValidation(t *testing.T) {
	cases := []struct {
		name       string
		body       map[string]any
		wantStatus int
	}{
		{"missing order_id", map[string]any{"amount": 100, "currency": "RUB"}, http.StatusBadRequest},
		{"zero amount", map[string]any{"order_id": "x", "amount": 0, "currency": "RUB"}, http.StatusBadRequest},
		{"wrong currency", map[string]any{"order_id": "x", "amount": 100, "currency": "USD"}, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, _ := json.Marshal(tc.body)
			resp, err := http.Post(baseURL+"/payments", "application/json", bytes.NewReader(b))
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("expected %d, got %d", tc.wantStatus, resp.StatusCode)
			}
		})
	}
}

package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type payment struct {
	ProviderPaymentID string `json:"provider_payment_id"`
	PaymentID         string `json:"payment_id"`
	OrderID           string `json:"order_id"`
	Amount            int64  `json:"amount"`
	Currency          string `json:"currency"`
	Description       string `json:"description"`
	Status            string `json:"status"`
	PaymentURL        string `json:"payment_url"`
}

type store struct {
	mu       sync.RWMutex
	payments map[string]payment
}

func main() {
	app := fiber.New(fiber.Config{AppName: "mock-payment-provider"})
	store := &store{payments: make(map[string]payment)}
	secret := env("WEBHOOK_SECRET", "local-secret")
	webhookTarget := env("WEBHOOK_TARGET_URL", "")
	httpClient := &http.Client{Timeout: 5 * time.Second}

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	app.Post("/payments", func(c *fiber.Ctx) error {
		var req payment
		if err := c.BodyParser(&req); err != nil {
			return fiber.ErrBadRequest
		}
		providerID := "provider_" + randomID()
		req.ProviderPaymentID = providerID
		req.Status = "pending"
		req.PaymentURL = "http://localhost:8082/pay/" + providerID

		store.mu.Lock()
		store.payments[providerID] = req
		store.mu.Unlock()

		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"provider_payment_id": providerID,
			"payment_url":         req.PaymentURL,
		})
	})

	app.Get("/payments/:id", func(c *fiber.Ctx) error {
		store.mu.RLock()
		p, ok := store.payments[c.Params("id")]
		store.mu.RUnlock()
		if !ok {
			return fiber.ErrNotFound
		}
		return c.JSON(fiber.Map{"status": p.Status})
	})

	app.Post("/payments/:id/refund", func(c *fiber.Ctx) error {
		store.mu.Lock()
		p, ok := store.payments[c.Params("id")]
		if ok {
			p.Status = "refunded"
			store.payments[p.ProviderPaymentID] = p
		}
		store.mu.Unlock()
		if !ok {
			return fiber.ErrNotFound
		}
		return c.JSON(fiber.Map{"status": "refunded"})
	})

	app.Post("/payments/:id/succeed", func(c *fiber.Ctx) error {
		store.mu.Lock()
		p, ok := store.payments[c.Params("id")]
		if ok {
			p.Status = "paid"
			store.payments[p.ProviderPaymentID] = p
		}
		store.mu.Unlock()
		if !ok {
			return fiber.ErrNotFound
		}

		body := map[string]string{
			"event_id":            "evt_" + randomID(),
			"event":               "payment.succeeded",
			"provider_payment_id": p.ProviderPaymentID,
		}
		data, _ := json.Marshal(body)
		signature := sign(secret, data)

		deliveryStatus := "not_sent"
		if webhookTarget != "" {
			req, err := http.NewRequestWithContext(c.UserContext(), http.MethodPost, webhookTarget, bytes.NewReader(data))
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Provider-Signature", signature)
			res, err := httpClient.Do(req)
			if err != nil {
				log.Printf("webhook delivery failed: %v", err)
				deliveryStatus = "failed"
			} else {
				_ = res.Body.Close()
				if res.StatusCode >= http.StatusBadRequest {
					log.Printf("webhook delivery returned status %d", res.StatusCode)
					deliveryStatus = "failed"
				} else {
					deliveryStatus = "sent"
				}
			}
		}

		return c.JSON(fiber.Map{
			"webhook_body":      json.RawMessage(data),
			"webhook_signature": signature,
			"delivery_status":   deliveryStatus,
		})
	})

	log.Fatal(app.Listen(env("PROVIDER_HTTP_ADDR", ":8081")))
}

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func randomID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", len(b))
	}
	return hex.EncodeToString(b[:])
}

func env(key string, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

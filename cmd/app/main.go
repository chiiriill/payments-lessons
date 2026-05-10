package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type CreatePaymentRequest struct {
	OrderID     string `json:"order_id"`
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
}

type Payment struct {
	ID          string `json:"id"`
	OrderID     string `json:"order_id"`
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
	Status      string `json:"status"`
	PaymentURL  string `json:"payment_url"`
}

type MemoryStore struct {
	mu       sync.RWMutex
	payments map[string]Payment
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{payments: make(map[string]Payment)}
}

func (s *MemoryStore) Save(payment Payment) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.payments[payment.ID] = payment
}

func (s *MemoryStore) Get(id string) (Payment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	payment, ok := s.payments[id]
	return payment, ok
}

func main() {
	store := NewMemoryStore()
	app := fiber.New()

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	app.Post("/payments", func(c *fiber.Ctx) error {
		var req CreatePaymentRequest
		if err := c.BodyParser(&req); err != nil {
			return fiber.ErrBadRequest
		}

		id := fmt.Sprintf("pay_%d", time.Now().UnixNano())
		payment := Payment{
			ID:          id,
			OrderID:     req.OrderID,
			Amount:      req.Amount,
			Currency:    req.Currency,
			Description: req.Description,
			Status:      "pending",
			PaymentURL:  "https://pay.local/" + id,
		}
		store.Save(payment)
		return c.Status(fiber.StatusCreated).JSON(payment)
	})

	app.Get("/payments/:id", func(c *fiber.Ctx) error {
		payment, ok := store.Get(c.Params("id"))
		if !ok {
			return fiber.ErrNotFound
		}
		return c.JSON(payment)
	})

	log.Fatal(app.Listen(":8080"))
}

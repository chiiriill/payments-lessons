package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

type CreatePaymentRequest struct {
	OrderID     string `json:"order_id"`
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
}

type CreatePaymentResponse struct {
	PaymentID  string `json:"payment_id"`
	PaymentURL string `json:"payment_url"`
}

func main() {
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
		return c.Status(fiber.StatusCreated).JSON(CreatePaymentResponse{
			PaymentID:  id,
			PaymentURL: "https://pay.local/" + id,
		})
	})

	log.Fatal(app.Listen(":8080"))
}

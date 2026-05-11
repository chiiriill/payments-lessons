package handler

import (
	"github.com/gofiber/fiber/v2"
	"stepik-payments-course/internal/infrastructure/http/middleware"
	"stepik-payments-course/internal/usecase/createPaymentUseCase"
)

type createPaymentRequest struct {
	OrderID     string `json:"order_id"`
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
}

func (h *Handler) CreatePayment(c *fiber.Ctx) error {
	var req createPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json body"})
	}
	if req.OrderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "order_id is required"})
	}
	if req.Amount <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "amount must be positive"})
	}
	if req.Currency != "RUB" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "only RUB currency is supported"})
	}

	log := middleware.LoggerFrom(c)

	payment, err := h.createPayment.Execute(c.Context(), createPaymentUseCase.Request{
		IdempotencyKey: c.Get("Idempotency-Key"),
		OrderID:        req.OrderID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Description:    req.Description,
	})
	if err != nil {
		return handleError(c, err)
	}
	log.InfoContext(c.Context(), "payment created",
		"payment_id", payment.ID,
		"order_id", payment.OrderID,
		"amount", payment.Amount,
	)
	return c.Status(fiber.StatusCreated).JSON(payment)
}

package httpapi

import (
	"errors"

	"stepik-payments-course/internal/payment"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *payment.Service
}

func NewHandler(service *payment.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(app *fiber.App) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Post("/payments", h.createPayment)
	app.Get("/payments/:id", h.getPayment)
	app.Post("/payments/:id/check", h.checkPayment)
	app.Post("/payments/:id/refund", h.refundPayment)
	app.Post("/webhooks/mock-provider", h.mockProviderWebhook)
}

type createPaymentRequest struct {
	OrderID     string `json:"order_id"`
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
}

func (h *Handler) createPayment(c *fiber.Ctx) error {
	var req createPaymentRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}
	var key *string
	if value := c.Get("Idempotency-Key"); value != "" {
		key = &value
	}
	p, err := h.service.Create(c.Context(), payment.CreateRequest{
		OrderID: req.OrderID, Amount: req.Amount, Currency: req.Currency, Description: req.Description, IdempotencyKey: key,
	})
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(p)
}

func (h *Handler) getPayment(c *fiber.Ctx) error {
	p, err := h.service.Get(c.Context(), c.Params("id"))
	if errors.Is(err, payment.ErrNotFound) {
		return fiber.ErrNotFound
	}
	if err != nil {
		return err
	}
	return c.JSON(p)
}

func (h *Handler) checkPayment(c *fiber.Ctx) error {
	p, err := h.service.Check(c.Context(), c.Params("id"))
	if err != nil {
		return err
	}
	return c.JSON(p)
}

func (h *Handler) refundPayment(c *fiber.Ctx) error {
	p, err := h.service.Refund(c.Context(), c.Params("id"))
	if errors.Is(err, payment.ErrInvalidTransition) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}
	if err != nil {
		return err
	}
	return c.JSON(p)
}

func (h *Handler) mockProviderWebhook(c *fiber.Ctx) error {
	type webhook struct {
		PaymentID string `json:"payment_id"`
		Event     string `json:"event"`
	}
	var req webhook
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}
	if req.Event != "payment.succeeded" {
		return c.JSON(fiber.Map{"status": "ignored"})
	}
	_, err := h.service.MarkPaid(c.Context(), req.PaymentID)
	if errors.Is(err, payment.ErrInvalidTransition) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

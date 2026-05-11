package handler

import (
	"github.com/gofiber/fiber/v2"
	"stepik-payments-course/internal/usecase/receiveWebhookUseCase"
)

func (h *Handler) ReceiveWebhook(c *fiber.Ctx) error {
	var req struct {
		PaymentID string `json:"payment_id"`
		Event     string `json:"event"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json body"})
	}
	if err := h.receiveWebhook.Execute(c.Context(), receiveWebhookUseCase.Request{
		PaymentID: req.PaymentID,
		Event:     req.Event,
	}); err != nil {
		return handleError(c, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

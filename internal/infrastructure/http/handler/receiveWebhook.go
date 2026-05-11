package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"stepik-payments-course/internal/usecase/receiveWebhookUseCase"
)

func (h *Handler) ReceiveWebhook(c *fiber.Ctx) error {
	body := c.Body()

	mac := hmac.New(sha256.New, []byte(h.webhookSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(c.Get("X-Provider-Signature"))) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid signature"})
	}

	var req struct {
		ProviderPaymentID string `json:"provider_payment_id"`
		Event             string `json:"event"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid json body"})
	}
	if err := h.receiveWebhook.Execute(c.Context(), receiveWebhookUseCase.Request{
		ProviderPaymentID: req.ProviderPaymentID,
		Event:             req.Event,
	}); err != nil {
		return handleError(c, err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

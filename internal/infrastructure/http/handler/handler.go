package handler

import (
	"context"
	"errors"

	"github.com/gofiber/fiber/v2"
	"stepik-payments-course/internal/common/apperror"
	"stepik-payments-course/internal/usecase/checkPaymentUseCase"
	"stepik-payments-course/internal/usecase/createPaymentUseCase"
	"stepik-payments-course/internal/usecase/getPaymentUseCase"
	"stepik-payments-course/internal/usecase/receiveWebhookUseCase"
	"stepik-payments-course/internal/usecase/refundPaymentUseCase"
)

type Handler struct {
	createPayment  *createPaymentUseCase.UseCase
	getPayment     *getPaymentUseCase.UseCase
	checkPayment   *checkPaymentUseCase.UseCase
	refundPayment  *refundPaymentUseCase.UseCase
	receiveWebhook *receiveWebhookUseCase.UseCase
	webhookSecret  string
	ping           func(context.Context) error
}

func New(
	createPayment *createPaymentUseCase.UseCase,
	getPayment *getPaymentUseCase.UseCase,
	checkPayment *checkPaymentUseCase.UseCase,
	refundPayment *refundPaymentUseCase.UseCase,
	receiveWebhook *receiveWebhookUseCase.UseCase,
	webhookSecret string,
	ping func(context.Context) error,
) *Handler {
	return &Handler{
		createPayment:  createPayment,
		getPayment:     getPayment,
		checkPayment:   checkPayment,
		refundPayment:  refundPayment,
		receiveWebhook: receiveWebhook,
		webhookSecret:  webhookSecret,
		ping:           ping,
	}
}

func (h *Handler) Register(app *fiber.App) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/readiness", func(c *fiber.Ctx) error {
		if err := h.ping(c.Context()); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "unavailable",
				"error":  "database unreachable",
			})
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})
	app.Post("/payments", h.CreatePayment)
	app.Get("/payments/:id", h.GetPayment)
	app.Post("/payments/:id/check", h.CheckPayment)
	app.Post("/payments/:id/refund", h.RefundPayment)
	app.Post("/webhooks/mock-provider", h.ReceiveWebhook)
}

func handleError(c *fiber.Ctx, err error) error {
	if errors.Is(err, apperror.ErrNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	if errors.Is(err, apperror.ErrInvalidTransition) {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}
	if errors.Is(err, apperror.ErrProviderTemporary) {
		return c.Status(fiber.StatusBadGateway).JSON(fiber.Map{"error": "payment provider unavailable"})
	}
	if errors.Is(err, apperror.ErrProviderPermanent) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "payment provider rejected request"})
	}
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
}

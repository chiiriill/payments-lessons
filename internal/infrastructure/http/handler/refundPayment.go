package handler

import (
	"github.com/gofiber/fiber/v2"
	"stepik-payments-course/internal/usecase/refundPaymentUseCase"
)

func (h *Handler) RefundPayment(c *fiber.Ctx) error {
	payment, err := h.refundPayment.Execute(c.Context(), refundPaymentUseCase.Request{
		PaymentID: c.Params("id"),
	})
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(payment)
}

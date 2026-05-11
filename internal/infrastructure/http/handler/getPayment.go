package handler

import (
	"github.com/gofiber/fiber/v2"
	"stepik-payments-course/internal/usecase/getPaymentUseCase"
)

func (h *Handler) GetPayment(c *fiber.Ctx) error {
	payment, err := h.getPayment.Execute(c.Context(), getPaymentUseCase.Request{
		PaymentID: c.Params("id"),
	})
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(payment)
}

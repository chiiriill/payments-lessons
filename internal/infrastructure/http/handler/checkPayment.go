package handler

import (
	"github.com/gofiber/fiber/v2"
	"stepik-payments-course/internal/usecase/checkPaymentUseCase"
)

func (h *Handler) CheckPayment(c *fiber.Ctx) error {
	payment, err := h.checkPayment.Execute(c.Context(), checkPaymentUseCase.Request{
		PaymentID: c.Params("id"),
	})
	if err != nil {
		return handleError(c, err)
	}
	return c.JSON(payment)
}

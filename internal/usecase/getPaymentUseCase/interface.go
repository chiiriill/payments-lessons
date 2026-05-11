package getPaymentUseCase

import (
	"context"
	"stepik-payments-course/internal/domain"
)

type repo interface {
	GetPaymentForGetPayment(ctx context.Context, paymentID string) (domain.Payment, error)
}

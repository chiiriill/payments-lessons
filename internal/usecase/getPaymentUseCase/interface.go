package getPaymentUseCase

import "context"

type repo interface {
	GetPaymentForGetPayment(ctx context.Context, paymentID string) (Payment, error)
}

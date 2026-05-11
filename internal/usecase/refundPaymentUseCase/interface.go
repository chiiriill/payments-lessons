package refundPaymentUseCase

import "context"

type repo interface {
	GetPaymentForRefundPayment(ctx context.Context, paymentID string) (Payment, error)
	SetRefundedForRefundPayment(ctx context.Context, paymentID string) (Payment, error)
}

type provider interface {
	RefundPayment(ctx context.Context, req ProviderRefundRequest) error
}

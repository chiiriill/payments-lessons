package refundPaymentUseCase

import (
	"context"
	"stepik-payments-course/internal/domain"
)

type repo interface {
	GetPaymentForRefundPayment(ctx context.Context, paymentID string) (domain.Payment, error)
	SetRefundedForRefundPayment(ctx context.Context, paymentID string) (domain.Payment, error)
}

type provider interface {
	RefundPayment(ctx context.Context, req ProviderRefundRequest) error
}

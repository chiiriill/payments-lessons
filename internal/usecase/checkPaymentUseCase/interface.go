package checkPaymentUseCase

import (
	"context"
	"stepik-payments-course/internal/domain"
)

type repo interface {
	GetPaymentForCheckPayment(ctx context.Context, paymentID string) (domain.Payment, error)
	SetPaidForCheckPayment(ctx context.Context, paymentID string) (domain.Payment, error)
}

type provider interface {
	CheckPayment(ctx context.Context, providerPaymentID string) (ProviderStatusResponse, error)
}

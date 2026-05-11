package checkPaymentUseCase

import "context"

type repo interface {
	GetPaymentForCheckPayment(ctx context.Context, paymentID string) (Payment, error)
	SetPaidForCheckPayment(ctx context.Context, paymentID string) (Payment, error)
}

type provider interface {
	CheckPayment(ctx context.Context, providerPaymentID string) (ProviderStatusResponse, error)
}

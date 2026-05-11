package createPaymentUseCase

import "context"

type repo interface {
	CreatePaymentByIdempotencyKey(ctx context.Context, req Request) (Payment, bool, error)
	SetProviderDataForCreatePayment(ctx context.Context, paymentID string, providerPaymentID string, paymentURL string) (Payment, error)
}

type provider interface {
	CreatePayment(ctx context.Context, req ProviderCreateRequest) (ProviderCreateResponse, error)
}

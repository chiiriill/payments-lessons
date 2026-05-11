package createPaymentUseCase

import (
	"context"
	"stepik-payments-course/internal/domain"
)

type repo interface {
	CreatePaymentByIdempotencyKey(ctx context.Context, req Request) (domain.Payment, bool, error)
	SetProviderDataForCreatePayment(ctx context.Context, paymentID string, providerPaymentID string, paymentURL string) (domain.Payment, error)
	SetPaymentFailedForCreatePayment(ctx context.Context, paymentID string) error
}

type provider interface {
	CreatePayment(ctx context.Context, req ProviderCreateRequest) (ProviderCreateResponse, error)
}

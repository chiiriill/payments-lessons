package receiveWebhookUseCase

import (
	"context"
	"stepik-payments-course/internal/domain"
)

type repo interface {
	GetPaymentByProviderIDForReceiveWebhook(ctx context.Context, providerPaymentID string) (domain.Payment, error)
	SetPaidForReceiveWebhook(ctx context.Context, paymentID string) (domain.Payment, error)
	SetFailedForReceiveWebhook(ctx context.Context, paymentID string) (domain.Payment, error)
}

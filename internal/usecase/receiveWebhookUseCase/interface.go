package receiveWebhookUseCase

import (
	"context"
	"stepik-payments-course/internal/domain"
)

type repo interface {
	MarkEventProcessedForReceiveWebhook(ctx context.Context, eventID string) (bool, error)
	GetPaymentByProviderIDForReceiveWebhook(ctx context.Context, providerPaymentID string) (domain.Payment, error)
	SetPaidForReceiveWebhook(ctx context.Context, paymentID string) (domain.Payment, error)
	SetFailedForReceiveWebhook(ctx context.Context, paymentID string) (domain.Payment, error)
}

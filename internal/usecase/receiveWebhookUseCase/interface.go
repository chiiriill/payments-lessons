package receiveWebhookUseCase

import "context"

type repo interface {
	GetPaymentForReceiveWebhook(ctx context.Context, paymentID string) (Payment, error)
	SetPaidForReceiveWebhook(ctx context.Context, paymentID string) (Payment, error)
}

package receiveWebhookUseCase

import (
	"context"
	"stepik-payments-course/internal/domain"
)

type repo interface {
	GetPaymentForReceiveWebhook(ctx context.Context, paymentID string) (domain.Payment, error)
	SetPaidForReceiveWebhook(ctx context.Context, paymentID string) (domain.Payment, error)
}

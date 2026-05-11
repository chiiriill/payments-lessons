package processStalePaymentsUseCase

import (
	"context"
	"time"

	"stepik-payments-course/internal/domain"
)

type repo interface {
	GetStalePendingPaymentsForWorker(ctx context.Context, olderThan time.Duration) ([]domain.Payment, error)
	SetPaidForWorker(ctx context.Context, paymentID string) (domain.Payment, error)
}

type provider interface {
	CheckPaymentForWorker(ctx context.Context, providerPaymentID string) (ProviderStatusResponse, error)
}

package processStalePaymentsUseCase

import (
	"context"
	"log/slog"
	"time"

	"stepik-payments-course/internal/common/status"
)

const staleDuration = 5 * time.Minute

type UseCase struct {
	repo     repo
	provider provider
	logger   *slog.Logger
}

func New(repo repo, provider provider, logger *slog.Logger) *UseCase {
	return &UseCase{repo: repo, provider: provider, logger: logger}
}

func (u *UseCase) Execute(ctx context.Context) Result {
	payments, err := u.repo.GetStalePendingPaymentsForWorker(ctx, staleDuration)
	if err != nil {
		u.logger.ErrorContext(ctx, "get stale payments failed", "error", err)
		return Result{}
	}

	var result Result
	for _, payment := range payments {
		providerRes, err := u.provider.CheckPaymentForWorker(ctx, payment.ProviderPaymentID)
		if err != nil {
			u.logger.ErrorContext(ctx, "provider check failed",
				"payment_id", payment.ID,
				"error", err,
			)
			result.Failed++
			continue
		}
		if providerRes.Status == status.Paid {
			if _, err := u.repo.SetPaidForWorker(ctx, payment.ID); err != nil {
				u.logger.ErrorContext(ctx, "set paid failed",
					"payment_id", payment.ID,
					"error", err,
				)
				result.Failed++
				continue
			}
		}
		result.Processed++
	}

	if result.Processed > 0 || result.Failed > 0 {
		u.logger.InfoContext(ctx, "stale payments processed",
			"processed", result.Processed,
			"failed", result.Failed,
		)
	}
	return result
}

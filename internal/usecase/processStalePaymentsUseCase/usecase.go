package processStalePaymentsUseCase

import (
	"context"
	"log/slog"
	"time"

	"stepik-payments-course/internal/common/status"
)

const (
	maxRetries = 3
	retryBase  = 200 * time.Millisecond
)

type UseCase struct {
	repo          repo
	provider      provider
	logger        *slog.Logger
	staleDuration time.Duration
}

func New(repo repo, provider provider, logger *slog.Logger, staleDuration time.Duration) *UseCase {
	return &UseCase{repo: repo, provider: provider, logger: logger, staleDuration: staleDuration}
}

func (u *UseCase) Execute(ctx context.Context) Result {
	payments, err := u.repo.GetStalePendingPaymentsForWorker(ctx, u.staleDuration)
	if err != nil {
		u.logger.ErrorContext(ctx, "get stale payments failed", "error", err)
		return Result{}
	}

	var result Result
	for _, payment := range payments {
		providerRes, err := u.checkWithRetry(ctx, payment.ProviderPaymentID)
		if err != nil {
			u.logger.ErrorContext(ctx, "provider check failed after retries",
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

func (u *UseCase) checkWithRetry(ctx context.Context, providerPaymentID string) (ProviderStatusResponse, error) {
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryBase * time.Duration(1<<attempt)
			select {
			case <-ctx.Done():
				return ProviderStatusResponse{}, ctx.Err()
			case <-time.After(delay):
			}
		}
		res, err := u.provider.CheckPaymentForWorker(ctx, providerPaymentID)
		if err == nil {
			return res, nil
		}
		lastErr = err
		u.logger.WarnContext(ctx, "provider check attempt failed",
			"provider_payment_id", providerPaymentID,
			"attempt", attempt+1,
			"error", err,
		)
	}
	return ProviderStatusResponse{}, lastErr
}

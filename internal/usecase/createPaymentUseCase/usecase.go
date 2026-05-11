package createPaymentUseCase

import (
	"context"
	"log/slog"

	"stepik-payments-course/internal/common/status"
	"stepik-payments-course/internal/domain"
)

type UseCase struct {
	repo     repo
	provider provider
	logger   *slog.Logger
}

func New(repo repo, provider provider, logger *slog.Logger) *UseCase {
	return &UseCase{repo: repo, provider: provider, logger: logger}
}

func (u *UseCase) Execute(ctx context.Context, req Request) (domain.Payment, error) {
	payment, existed, err := u.repo.CreatePaymentByIdempotencyKey(ctx, req)
	if err != nil {
		return domain.Payment{}, err
	}
	if existed {
		if payment.ProviderPaymentID != "" || status.IsTerminal(payment.Status) {
			u.logger.InfoContext(ctx, "idempotent create - returning existing payment",
				"idempotency_key", req.IdempotencyKey,
				"payment_id", payment.ID,
				"status", payment.Status,
			)
			return payment, nil
		}
		u.logger.InfoContext(ctx, "idempotent create - retrying provider call after partial failure",
			"idempotency_key", req.IdempotencyKey,
			"payment_id", payment.ID,
		)
	}

	providerRes, err := u.provider.CreatePayment(ctx, ProviderCreateRequest{
		PaymentID:   payment.ID,
		OrderID:     payment.OrderID,
		Amount:      payment.Amount,
		Currency:    payment.Currency,
		Description: payment.Description,
	})
	if err != nil {
		if setErr := u.repo.SetPaymentFailedForCreatePayment(ctx, payment.ID); setErr != nil {
			u.logger.Error("mark payment failed after provider error", "payment_id", payment.ID, "error", setErr)
		}
		return domain.Payment{}, err
	}

	return u.repo.SetProviderDataForCreatePayment(ctx, payment.ID, providerRes.ProviderPaymentID, providerRes.PaymentURL)
}

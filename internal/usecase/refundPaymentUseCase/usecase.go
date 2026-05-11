package refundPaymentUseCase

import (
	"context"

	"stepik-payments-course/internal/common/apperror"
	"stepik-payments-course/internal/common/status"
)

type UseCase struct {
	repo     repo
	provider provider
}

func New(repo repo, provider provider) *UseCase {
	return &UseCase{repo: repo, provider: provider}
}

func (u *UseCase) Execute(ctx context.Context, req Request) (Payment, error) {
	payment, err := u.repo.GetPaymentForRefundPayment(ctx, req.PaymentID)
	if err != nil {
		return Payment{}, err
	}
	if payment.Status != status.Paid {
		return Payment{}, apperror.ErrInvalidTransition
	}

	if payment.ProviderPaymentID != "" {
		if err := u.provider.RefundPayment(ctx, ProviderRefundRequest{
			ProviderPaymentID: payment.ProviderPaymentID,
			Amount:            payment.Amount,
		}); err != nil {
			return Payment{}, err
		}
	}

	return u.repo.SetRefundedForRefundPayment(ctx, payment.ID)
}

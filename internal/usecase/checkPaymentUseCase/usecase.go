package checkPaymentUseCase

import (
	"context"

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
	payment, err := u.repo.GetPaymentForCheckPayment(ctx, req.PaymentID)
	if err != nil {
		return Payment{}, err
	}
	if payment.ProviderPaymentID == "" {
		return payment, nil
	}

	providerRes, err := u.provider.CheckPayment(ctx, payment.ProviderPaymentID)
	if err != nil {
		return Payment{}, err
	}
	if providerRes.Status == status.Paid && payment.Status == status.Pending {
		return u.repo.SetPaidForCheckPayment(ctx, payment.ID)
	}
	return payment, nil
}

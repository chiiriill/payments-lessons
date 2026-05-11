package createPaymentUseCase

import (
	"context"
	"stepik-payments-course/internal/domain"
)

type UseCase struct {
	repo     repo
	provider provider
}

func New(repo repo, provider provider) *UseCase {
	return &UseCase{repo: repo, provider: provider}
}

func (u *UseCase) Execute(ctx context.Context, req Request) (domain.Payment, error) {
	payment, existed, err := u.repo.CreatePaymentByIdempotencyKey(ctx, req)
	if err != nil {
		return domain.Payment{}, err
	}
	if existed {
		return payment, nil
	}

	providerRes, err := u.provider.CreatePayment(ctx, ProviderCreateRequest{
		PaymentID:   payment.ID,
		OrderID:     payment.OrderID,
		Amount:      payment.Amount,
		Currency:    payment.Currency,
		Description: payment.Description,
	})
	if err != nil {
		return domain.Payment{}, err
	}

	return u.repo.SetProviderDataForCreatePayment(ctx, payment.ID, providerRes.ProviderPaymentID, providerRes.PaymentURL)
}

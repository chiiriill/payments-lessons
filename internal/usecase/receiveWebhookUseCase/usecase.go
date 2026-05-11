package receiveWebhookUseCase

import (
	"context"

	"stepik-payments-course/internal/common/status"
)

type UseCase struct {
	repo repo
}

func New(repo repo) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) Execute(ctx context.Context, req Request) error {
	if req.Event != "payment.succeeded" {
		return nil
	}
	payment, err := u.repo.GetPaymentByProviderIDForReceiveWebhook(ctx, req.ProviderPaymentID)
	if err != nil {
		return err
	}
	if !status.CanTransition(payment.Status, status.Paid) {
		return nil
	}
	_, err = u.repo.SetPaidForReceiveWebhook(ctx, payment.ID)
	return err
}

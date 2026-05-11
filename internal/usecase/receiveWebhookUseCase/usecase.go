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
	var targetStatus string
	switch req.Event {
	case "payment.succeeded":
		targetStatus = status.Paid
	case "payment.failed":
		targetStatus = status.Failed
	default:
		return nil
	}

	payment, err := u.repo.GetPaymentByProviderIDForReceiveWebhook(ctx, req.ProviderPaymentID)
	if err != nil {
		return err
	}
	if !status.CanTransition(payment.Status, targetStatus) {
		return nil
	}
	if targetStatus == status.Paid {
		_, err = u.repo.SetPaidForReceiveWebhook(ctx, payment.ID)
	} else {
		_, err = u.repo.SetFailedForReceiveWebhook(ctx, payment.ID)
	}
	return err
}

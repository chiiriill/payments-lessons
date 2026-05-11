package getPaymentUseCase

import (
	"context"
	"stepik-payments-course/internal/domain"
)

type UseCase struct {
	repo repo
}

func New(repo repo) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) Execute(ctx context.Context, req Request) (domain.Payment, error) {
	return u.repo.GetPaymentForGetPayment(ctx, req.PaymentID)
}

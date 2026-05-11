package getPaymentUseCase

import "context"

type UseCase struct {
	repo repo
}

func New(repo repo) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) Execute(ctx context.Context, req Request) (Payment, error) {
	return u.repo.GetPaymentForGetPayment(ctx, req.PaymentID)
}

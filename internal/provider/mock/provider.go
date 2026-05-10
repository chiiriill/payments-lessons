package mock

import (
	"context"

	"stepik-payments-course/internal/payment"
)

type Provider struct{}

func NewProvider() *Provider {
	return &Provider{}
}

func (p *Provider) CreatePayment(ctx context.Context, pay payment.Payment) (string, string, error) {
	providerID := "provider_" + pay.ID
	return providerID, "https://mock-provider.local/pay/" + providerID, nil
}

func (p *Provider) CheckPayment(ctx context.Context, providerPaymentID string) (string, error) {
	return payment.StatusPending, nil
}

func (p *Provider) RefundPayment(ctx context.Context, providerPaymentID string, amount int64) error {
	return nil
}

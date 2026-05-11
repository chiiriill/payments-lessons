package mock

import (
	"context"

	"stepik-payments-course/internal/common/status"
	"stepik-payments-course/internal/usecase/checkPaymentUseCase"
	"stepik-payments-course/internal/usecase/createPaymentUseCase"
	"stepik-payments-course/internal/usecase/refundPaymentUseCase"
)

type Provider struct{}

func NewProvider() *Provider {
	return &Provider{}
}

func (p *Provider) CreatePayment(_ context.Context, req createPaymentUseCase.ProviderCreateRequest) (createPaymentUseCase.ProviderCreateResponse, error) {
	providerID := "provider_" + req.PaymentID
	return createPaymentUseCase.ProviderCreateResponse{
		ProviderPaymentID: providerID,
		PaymentURL:        "https://mock-provider.local/pay/" + providerID,
	}, nil
}

func (p *Provider) CheckPayment(_ context.Context, _ string) (checkPaymentUseCase.ProviderStatusResponse, error) {
	return checkPaymentUseCase.ProviderStatusResponse{Status: status.Pending}, nil
}

func (p *Provider) RefundPayment(_ context.Context, _ refundPaymentUseCase.ProviderRefundRequest) error {
	return nil
}

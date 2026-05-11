package mock

import (
	"context"
	"time"

	"stepik-payments-course/internal/common/status"
	"stepik-payments-course/internal/usecase/checkPaymentUseCase"
	"stepik-payments-course/internal/usecase/createPaymentUseCase"
	"stepik-payments-course/internal/usecase/refundPaymentUseCase"
)

type Provider struct {
	baseURL string
	timeout time.Duration
}

func NewProvider(baseURL string, timeout time.Duration) *Provider {
	return &Provider{baseURL: baseURL, timeout: timeout}
}

func (p *Provider) CreatePayment(_ context.Context, req createPaymentUseCase.ProviderCreateRequest) (createPaymentUseCase.ProviderCreateResponse, error) {
	providerID := "provider_" + req.PaymentID
	return createPaymentUseCase.ProviderCreateResponse{
		ProviderPaymentID: providerID,
		PaymentURL:        p.baseURL + "/pay/" + providerID,
	}, nil
}

func (p *Provider) CheckPayment(_ context.Context, _ string) (checkPaymentUseCase.ProviderStatusResponse, error) {
	return checkPaymentUseCase.ProviderStatusResponse{Status: status.Pending}, nil
}

func (p *Provider) RefundPayment(_ context.Context, _ refundPaymentUseCase.ProviderRefundRequest) error {
	return nil
}

package refundPaymentUseCase

type Request struct {
	PaymentID string
}

type ProviderRefundRequest struct {
	ProviderPaymentID string
	Amount            int64
}

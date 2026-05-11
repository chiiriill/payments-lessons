package checkPaymentUseCase

type Request struct {
	PaymentID string
}

type ProviderStatusResponse struct {
	Status string
}

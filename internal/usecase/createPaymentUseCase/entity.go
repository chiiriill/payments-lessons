package createPaymentUseCase

type Request struct {
	IdempotencyKey string
	OrderID        string
	Amount         int64
	Currency       string
	Description    string
}

type ProviderCreateRequest struct {
	PaymentID   string
	OrderID     string
	Amount      int64
	Currency    string
	Description string
}

type ProviderCreateResponse struct {
	ProviderPaymentID string
	PaymentURL        string
}

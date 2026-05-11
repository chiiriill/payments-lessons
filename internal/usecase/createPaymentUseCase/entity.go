package createPaymentUseCase

import "time"

type Request struct {
	IdempotencyKey string
	OrderID        string
	Amount         int64
	Currency       string
	Description    string
}

type Payment struct {
	ID                string    `json:"id"`
	OrderID           string    `json:"order_id"`
	Amount            int64     `json:"amount"`
	Currency          string    `json:"currency"`
	Description       string    `json:"description"`
	Status            string    `json:"status"`
	ProviderPaymentID string    `json:"provider_payment_id"`
	PaymentURL        string    `json:"payment_url"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
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

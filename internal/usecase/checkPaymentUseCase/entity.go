package checkPaymentUseCase

import "time"

type Request struct {
	PaymentID string
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

type ProviderStatusResponse struct {
	Status string
}

package receiveWebhookUseCase

type Request struct {
	PaymentID string
	Event     string
}

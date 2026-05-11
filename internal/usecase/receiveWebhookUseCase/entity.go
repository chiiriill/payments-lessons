package receiveWebhookUseCase

type Request struct {
	EventID           string
	ProviderPaymentID string
	Event             string
}

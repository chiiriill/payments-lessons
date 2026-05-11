package receiveWebhookUseCase

type Request struct {
	ProviderPaymentID string
	Event             string
}

package processStalePaymentsUseCase

type Result struct {
	Processed int
	Failed    int
}

type ProviderStatusResponse struct {
	Status string
}

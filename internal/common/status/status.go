package status

const (
	Pending  = "pending"
	Paid     = "paid"
	Failed   = "failed"
	Refunded = "refunded"
)

// allowedTransitions defines the valid state machine for payments.
// A transition not listed here is forbidden (ErrInvalidTransition).
var allowedTransitions = map[string][]string{
	Pending: {Paid, Failed},
	Paid:    {Refunded},
	// Failed and Refunded are terminal: no outgoing transitions.
}

func CanTransition(from, to string) bool {
	for _, allowed := range allowedTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

func IsTerminal(s string) bool {
	_, hasTransitions := allowedTransitions[s]
	return !hasTransitions
}

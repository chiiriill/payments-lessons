package receiveWebhookUseCase

import (
	"context"
	"log/slog"

	"stepik-payments-course/internal/common/status"
	"stepik-payments-course/internal/events"
)

type UseCase struct {
	repo      repo
	publisher publisher
	logger    *slog.Logger
}

func New(repo repo, publisher publisher, logger *slog.Logger) *UseCase {
	return &UseCase{repo: repo, publisher: publisher, logger: logger}
}

func (u *UseCase) Execute(ctx context.Context, req Request) error {
	var targetStatus string
	switch req.Event {
	case "payment.succeeded":
		targetStatus = status.Paid
	case "payment.failed":
		targetStatus = status.Failed
	default:
		u.logger.InfoContext(ctx, "webhook event ignored", "event_id", req.EventID, "event", req.Event)
		return nil
	}

	if req.EventID != "" {
		isNew, err := u.repo.MarkEventProcessedForReceiveWebhook(ctx, req.EventID)
		if err != nil {
			return err
		}
		if !isNew {
			u.logger.InfoContext(ctx, "duplicate webhook event ignored",
				"event_id", req.EventID,
				"event", req.Event,
			)
			return nil
		}
	}

	payment, err := u.repo.GetPaymentByProviderIDForReceiveWebhook(ctx, req.ProviderPaymentID)
	if err != nil {
		return err
	}
	if !status.CanTransition(payment.Status, targetStatus) {
		u.logger.InfoContext(ctx, "webhook status transition skipped",
			"event_id", req.EventID,
			"payment_id", payment.ID,
			"current_status", payment.Status,
			"target_status", targetStatus,
		)
		return nil
	}
	if targetStatus == status.Paid {
		if _, err = u.repo.SetPaidForReceiveWebhook(ctx, payment.ID); err != nil {
			return err
		}
		// DUAL WRITE: DB transaction committed above, but process may crash before Publish.
		// If Publish fails, the event is permanently lost — no downstream service learns the payment was paid.
		_ = u.publisher.Publish(ctx, events.Event{
			Type: "payment.paid",
			Payload: map[string]any{
				"payment_id": payment.ID,
				"order_id":   payment.OrderID,
				"amount":     payment.Amount,
			},
		})
		return nil
	}
	_, err = u.repo.SetFailedForReceiveWebhook(ctx, payment.ID)
	return err
}

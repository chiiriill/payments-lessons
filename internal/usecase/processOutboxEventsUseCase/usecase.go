package processOutboxEventsUseCase

import (
	"context"
	"log/slog"

	"stepik-payments-course/internal/events"
)

const batchSize = 100

type UseCase struct {
	repo      repo
	publisher publisher
	logger    *slog.Logger
}

func New(repo repo, publisher publisher, logger *slog.Logger) *UseCase {
	return &UseCase{repo: repo, publisher: publisher, logger: logger}
}

func (u *UseCase) Execute(ctx context.Context) {
	pending, err := u.repo.GetPendingOutboxEventsForWorker(ctx, batchSize)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to get pending outbox events", "error", err)
		return
	}

	for _, e := range pending {
		if err := u.publisher.Publish(ctx, events.Event{Type: e.EventType, Payload: e.Payload}); err != nil {
			u.logger.WarnContext(ctx, "failed to publish outbox event",
				"id", e.ID,
				"event_type", e.EventType,
				"error", err,
			)
			continue
		}
		if err := u.repo.MarkOutboxEventPublishedForWorker(ctx, e.ID); err != nil {
			u.logger.WarnContext(ctx, "failed to mark outbox event as published",
				"id", e.ID,
				"event_type", e.EventType,
				"error", err,
			)
		}
	}
}

package processOutboxEventsUseCase

import (
	"context"

	"stepik-payments-course/internal/events"
)

type repo interface {
	GetPendingOutboxEventsForWorker(ctx context.Context, limit int) ([]events.OutboxEvent, error)
	MarkOutboxEventPublishedForWorker(ctx context.Context, id int64) error
}

type publisher interface {
	Publish(ctx context.Context, event events.Event) error
}

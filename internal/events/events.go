package events

import (
	"context"
	"log/slog"
	"time"
)

type Event struct {
	Type    string
	Payload map[string]any
}

type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

type LogPublisher struct {
	logger *slog.Logger
}

func NewLogPublisher(logger *slog.Logger) *LogPublisher {
	return &LogPublisher{logger: logger}
}

func (p *LogPublisher) Publish(ctx context.Context, event Event) error {
	p.logger.InfoContext(ctx, "publishing event", "type", event.Type, "payload", event.Payload)
	return nil
}

type OutboxEvent struct {
	ID          int64
	EventType   string
	AggregateID string
	Payload     map[string]any
	CreatedAt   time.Time
	PublishedAt *time.Time
}

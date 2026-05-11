package events

import (
	"context"
	"log/slog"
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

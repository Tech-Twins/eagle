package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Publisher struct {
	client *redis.Client
}

func NewPublisher(client *redis.Client) *Publisher {
	return &Publisher{client: client}
}

func (p *Publisher) Publish(ctx context.Context, stream, eventType string, data any) error {
	event := Event{
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		Data:      data,
	}

	eventJSON, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	args := &redis.XAddArgs{
		Stream: stream,
		Values: map[string]any{
			"event": eventJSON,
		},
	}

	if _, err := p.client.XAdd(ctx, args).Result(); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

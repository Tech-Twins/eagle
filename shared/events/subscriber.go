package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type Handler func(ctx context.Context, event Event) error

type Subscriber struct {
	client        *redis.Client
	group         string
	consumer      string
	stream        string
	handler       Handler
	batchSize     int64
	blockDuration time.Duration
}

type SubscriberConfig struct {
	Group         string
	Consumer      string
	Stream        string
	Handler       Handler
	BatchSize     int64
	BlockDuration time.Duration
}

func NewSubscriber(client *redis.Client, config SubscriberConfig) *Subscriber {
	if config.BatchSize == 0 {
		config.BatchSize = 10
	}
	if config.BlockDuration == 0 {
		config.BlockDuration = 5 * time.Second
	}

	return &Subscriber{
		client:        client,
		group:         config.Group,
		consumer:      config.Consumer,
		stream:        config.Stream,
		handler:       config.Handler,
		batchSize:     config.BatchSize,
		blockDuration: config.BlockDuration,
	}
}

func (s *Subscriber) Start(ctx context.Context) error {
	// Create consumer group if it doesn't exist
	err := s.client.XGroupCreateMkStream(ctx, s.stream, s.group, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	log.Printf("Subscriber started: stream=%s, group=%s, consumer=%s", s.stream, s.group, s.consumer)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Subscriber stopping: %s", s.stream)
			return ctx.Err()
		default:
			if err := s.readMessages(ctx); err != nil {
				log.Printf("Error reading messages: %v", err)
				time.Sleep(time.Second)
			}
		}
	}
}

func (s *Subscriber) readMessages(ctx context.Context) error {
	streams, err := s.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    s.group,
		Consumer: s.consumer,
		Streams:  []string{s.stream, ">"},
		Count:    s.batchSize,
		Block:    s.blockDuration,
	}).Result()

	if err == redis.Nil {
		return nil // No messages
	}
	if err != nil {
		return fmt.Errorf("failed to read from stream: %w", err)
	}

	for _, stream := range streams {
		for _, message := range stream.Messages {
			if err := s.processMessage(ctx, message); err != nil {
				log.Printf("Failed to process message %s: %v", message.ID, err)
				// Don't ACK failed messages - they'll be retried
				continue
			}

			// ACK successful message
			if err := s.client.XAck(ctx, s.stream, s.group, message.ID).Err(); err != nil {
				log.Printf("Failed to ACK message %s: %v", message.ID, err)
			}
		}
	}

	return nil
}

func (s *Subscriber) processMessage(ctx context.Context, message redis.XMessage) error {
	eventData, ok := message.Values["event"].(string)
	if !ok {
		return fmt.Errorf("invalid message format")
	}

	var event Event
	if err := json.Unmarshal([]byte(eventData), &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	return s.handler(ctx, event)
}

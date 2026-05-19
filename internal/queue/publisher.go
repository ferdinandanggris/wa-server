package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wa-server/internal/models"
)

// Publisher sends messages to the RabbitMQ exchange.
type Publisher struct {
	rmq *RabbitMQ
}

// NewPublisher creates a new Publisher using the given RabbitMQ connection.
func NewPublisher(rmq *RabbitMQ) *Publisher {
	return &Publisher{rmq: rmq}
}

// PublishInbound sends an inbound message to the RabbitMQ exchange.
func (p *Publisher) PublishInbound(ctx context.Context, msg *models.Message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal inbound message: %w", err)
	}
	return p.rmq.Publish(ctx, RoutingKeyInbound, body)
}

// PublishOutbound sends an outbound message to the RabbitMQ exchange.
func (p *Publisher) PublishOutbound(ctx context.Context, msg *models.Message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal outbound message: %w", err)
	}
	return p.rmq.Publish(ctx, RoutingKeyOutbound, body)
}

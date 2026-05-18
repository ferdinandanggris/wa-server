package queue

import (
	"context"
	"encoding/json"

	"github.com/wa-server/internal/models"
)

type Publisher struct {
	rmq *RabbitMQ
}

func NewPublisher(rmq *RabbitMQ) *Publisher {
	return &Publisher{rmq: rmq}
}

func (p *Publisher) PublishInbound(ctx context.Context, msg *models.Message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.rmq.Publish(ctx, QueueInbound, body)
}

func (p *Publisher) PublishOutbound(ctx context.Context, msg *models.Message) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return p.rmq.Publish(ctx, QueueOutbound, body)
}

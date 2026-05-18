package queue

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/wa-server/internal/config"
)

const (
	ExchangeWhatsApp   = "whatsapp"
	QueueInbound       = "inbound_messages"
	QueueOutbound      = "outbound_messages"
	RoutingKeyInbound  = "inbound"
	RoutingKeyOutbound = "outbound"
)

type RabbitMQ struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	cfg     *config.RabbitMQConfig
	mu      sync.Mutex
}

func NewRabbitMQ(cfg *config.RabbitMQConfig) (*RabbitMQ, error) {
	conn, err := amqp.Dial(cfg.URL())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	rmq := &RabbitMQ{
		conn:    conn,
		channel: ch,
		cfg:     cfg,
	}

	if err := rmq.setup(); err != nil {
		rmq.Close()
		return nil, err
	}

	return rmq, nil
}

func (r *RabbitMQ) setup() error {
	err := r.channel.ExchangeDeclare(
		ExchangeWhatsApp,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	_, err = r.channel.QueueDeclare(
		QueueInbound,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare inbound queue: %w", err)
	}

	_, err = r.channel.QueueDeclare(
		QueueOutbound,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare outbound queue: %w", err)
	}

	err = r.channel.QueueBind(
		QueueInbound,
		RoutingKeyInbound,
		ExchangeWhatsApp,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind inbound queue: %w", err)
	}

	err = r.channel.QueueBind(
		QueueOutbound,
		RoutingKeyOutbound,
		ExchangeWhatsApp,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind outbound queue: %w", err)
	}

	return nil
}

func (r *RabbitMQ) Publish(ctx context.Context, queueName string, body []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	err := r.channel.PublishWithContext(ctx,
		ExchangeWhatsApp,
		queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	slog.Debug("published message", "queue", queueName, "size", len(body))
	return nil
}

func (r *RabbitMQ) Consume(ctx context.Context, queueName string) (<-chan amqp.Delivery, error) {
	msgs, err := r.channel.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start consuming: %w", err)
	}

	slog.Info("started consuming queue", "queue", queueName)
	return msgs, nil
}

func (r *RabbitMQ) Close() error {
	if r.channel != nil {
		r.channel.Close()
	}
	if r.conn != nil {
		return r.conn.Close()
	}
	return nil
}

func (r *RabbitMQ) Channel() *amqp.Channel {
	return r.channel
}

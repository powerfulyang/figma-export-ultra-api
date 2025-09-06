// Package mqx provides message queue functionality using RabbitMQ
package mqx

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/samber/lo"
)

// Publisher defines the interface for message publishing
type Publisher interface {
	Publish(ctx context.Context, routingKey string, body []byte) error
	Close() error
}

// RabbitPublisher implements Publisher interface using RabbitMQ
type RabbitPublisher struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	exchange string
}

// NewRabbitPublisher creates a new RabbitMQ publisher
func NewRabbitPublisher(url string, exchange string) (*RabbitPublisher, error) {
	exchange = lo.Ternary(exchange != "", exchange, "events")
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}
	return &RabbitPublisher{conn: conn, ch: ch, exchange: exchange}, nil
}

// Publish sends a message to the specified routing key
func (p *RabbitPublisher) Publish(ctx context.Context, routingKey string, body []byte) error {
	return p.ch.PublishWithContext(ctx, p.exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		Timestamp:    time.Now(),
		DeliveryMode: amqp.Persistent,
	})
}

// Close closes the RabbitMQ connection and channel
func (p *RabbitPublisher) Close() error {
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

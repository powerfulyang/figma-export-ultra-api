package mqx

import (
    "context"
    "time"

    amqp "github.com/rabbitmq/amqp091-go"
    "github.com/samber/lo"
)

type Publisher interface {
	Publish(ctx context.Context, routingKey string, body []byte) error
	Close() error
}

type RabbitPublisher struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	exchange string
}

func NewRabbitPublisher(url string, exchange string) (*RabbitPublisher, error) {
    exchange = lo.Ternary(exchange != "", exchange, "events")
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}
	return &RabbitPublisher{conn: conn, ch: ch, exchange: exchange}, nil
}

func (p *RabbitPublisher) Publish(ctx context.Context, routingKey string, body []byte) error {
	return p.ch.PublishWithContext(ctx, p.exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         body,
		Timestamp:    time.Now(),
		DeliveryMode: amqp.Persistent,
	})
}

func (p *RabbitPublisher) Close() error {
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

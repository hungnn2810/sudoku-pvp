package rabbitmq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher sends messages to the game.events topic exchange.
// Each Publisher owns exactly one AMQP channel — channels must never be shared
// across goroutines (T-03-05, Pitfall 2).
type Publisher struct {
	ch *amqp.Channel
}

// NewPublisher creates a Publisher by opening a dedicated channel from the connection.
// The Publisher takes ownership of the channel; call Close() when done.
func NewPublisher(conn *Connection) (*Publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("publisher open channel: %w", err)
	}
	return &Publisher{ch: ch}, nil
}

// Publish sends a message to the game.events exchange with the given routing key.
// The message body is treated as application/json.
// The context controls the publish timeout.
func (p *Publisher) Publish(ctx context.Context, routingKey string, body []byte) error {
	err := p.ch.PublishWithContext(
		ctx,
		ExchangeGameEvents, // exchange
		routingKey,         // routing key
		false,              // mandatory
		false,              // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return fmt.Errorf("publish to %s with key %s: %w", ExchangeGameEvents, routingKey, err)
	}
	return nil
}

// Close releases the underlying AMQP channel.
func (p *Publisher) Close() {
	if p.ch != nil {
		p.ch.Close()
	}
}

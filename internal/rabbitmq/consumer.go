package rabbitmq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const maxXDeathCount = 3

// Consumer receives deliveries from a named queue with manual Ack/Nack.
// Each Consumer owns exactly one AMQP channel — channels must never be shared
// across goroutines (T-03-05, Pitfall 2).
type Consumer struct {
	ch        *amqp.Channel
	queueName string
}

// NewConsumer creates a Consumer by opening a dedicated channel from the connection.
func NewConsumer(conn *Connection, queueName string) (*Consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("consumer open channel: %w", err)
	}
	return &Consumer{ch: ch, queueName: queueName}, nil
}

// Consume starts consuming messages from the queue and calls handler for each delivery.
// Manual Ack/Nack: on handler success → Ack; on handler error → Nack(requeue=false, routes to DLX).
// x-death guard: if x-death count >= 3, Ack and skip to prevent infinite DLX retry loops (T-03-04, Pitfall 8).
// The call blocks until ctx is cancelled or the channel closes.
// Goroutine leak prevention: the consumer loop exits when ctx is cancelled (T-03-03).
func (c *Consumer) Consume(ctx context.Context, handler func(amqp.Delivery) error) error {
	deliveries, err := c.ch.Consume(
		c.queueName,
		"",    // consumer tag (auto-generated)
		false, // auto-ack disabled — we ack/nack manually
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("consume from %s: %w", c.queueName, err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				// Channel closed (connection dropped).
				return fmt.Errorf("delivery channel closed for queue %s", c.queueName)
			}

			// x-death guard: prevent infinite retry loops (Pitfall 8, T-03-04).
			if xDeathCount(d) >= maxXDeathCount {
				// Message has been dead-lettered 3+ times — acknowledge and discard
				// (it was already routed to game.dead.queue by the DLX; Nack would loop).
				_ = d.Ack(false)
				continue
			}

			if err := handler(d); err != nil {
				// Nack without requeue — routes to game.dlx → game.dead.queue.
				_ = d.Nack(false, false)
			} else {
				_ = d.Ack(false)
			}
		}
	}
}

// xDeathCount extracts the total x-death rejection count from a delivery's headers.
// The x-death header is an array of tables; each table has a "count" field.
// Summing all count fields gives the total number of times this message has been dead-lettered.
func xDeathCount(d amqp.Delivery) int64 {
	raw, ok := d.Headers["x-death"]
	if !ok {
		return 0
	}

	entries, ok := raw.([]interface{})
	if !ok {
		return 0
	}

	var total int64
	for _, entry := range entries {
		table, ok := entry.(amqp.Table)
		if !ok {
			continue
		}
		if count, ok := table["count"]; ok {
			switch v := count.(type) {
			case int64:
				total += v
			case int32:
				total += int64(v)
			case int:
				total += int64(v)
			}
		}
	}
	return total
}

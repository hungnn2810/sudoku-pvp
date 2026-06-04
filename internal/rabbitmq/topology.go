package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Exchange and queue name constants prevent typo-based bugs across all consumers and publishers.
const (
	ExchangeGameEvents = "game.events"
	ExchangeDLX        = "game.dlx"
	QueueDead          = "game.dead.queue"
)

// queues defines all domain queues with their routing key bindings to game.events.
// Per DEC-012:
//   - ranking.queue: match.finished
//   - wallet.queue: match.finished
//   - mission.queue: mission.completed
//   - analytics.queue: "#" (wildcard — receives all events)
//   - notification.queue: ranking.changed
var queues = []struct {
	Name       string
	RoutingKey string
}{
	{"ranking.queue", "match.finished"},
	{"wallet.queue", "match.finished"},
	{"mission.queue", "mission.completed"},
	{"analytics.queue", "#"},
	{"notification.queue", "ranking.changed"},
}

// DeclareAll idempotently provisions the full RabbitMQ topology:
//  1. game.events topic exchange (main fanout exchange for all domain events)
//  2. game.dlx direct exchange (dead letter exchange)
//  3. game.dead.queue (terminal dead letter queue, bound to game.dlx)
//  4. All 5 domain queues with x-dead-letter-exchange set to game.dlx
//  5. Bindings from each domain queue to game.events via their routing keys
//
// Idempotent: safe to call multiple times on the same channel.
// Called at startup and after every reconnect so topology is always present.
func DeclareAll(ch *amqp.Channel) error {
	// 1. Declare game.events as a durable topic exchange.
	if err := ch.ExchangeDeclare(
		ExchangeGameEvents,
		amqp.ExchangeTopic,
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		return fmt.Errorf("declare game.events exchange: %w", err)
	}

	// 2. Declare game.dlx as a durable direct exchange.
	if err := ch.ExchangeDeclare(
		ExchangeDLX,
		amqp.ExchangeDirect,
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		return fmt.Errorf("declare game.dlx exchange: %w", err)
	}

	// 3. Declare game.dead.queue as a durable queue.
	if _, err := ch.QueueDeclare(
		QueueDead,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	); err != nil {
		return fmt.Errorf("declare game.dead.queue: %w", err)
	}

	// 4. Bind game.dead.queue to game.dlx with empty routing key (direct exchange receives all dead letters).
	if err := ch.QueueBind(QueueDead, "", ExchangeDLX, false, nil); err != nil {
		return fmt.Errorf("bind game.dead.queue to game.dlx: %w", err)
	}

	// 5. Declare all domain queues with DLX config and bind to game.events.
	for _, q := range queues {
		if _, err := ch.QueueDeclare(
			q.Name,
			true,  // durable
			false, // auto-delete
			false, // exclusive
			false, // no-wait
			amqp.Table{"x-dead-letter-exchange": ExchangeDLX},
		); err != nil {
			return fmt.Errorf("declare queue %s: %w", q.Name, err)
		}

		if err := ch.QueueBind(q.Name, q.RoutingKey, ExchangeGameEvents, false, nil); err != nil {
			return fmt.Errorf("bind queue %s to %s with key %s: %w",
				q.Name, ExchangeGameEvents, q.RoutingKey, err)
		}
	}

	return nil
}

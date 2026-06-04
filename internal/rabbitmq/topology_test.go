//go:build integration

package rabbitmq_test

import (
	"context"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"github.com/rs/zerolog"

	"sudoku-pvp/internal/rabbitmq"
	"sudoku-pvp/internal/testutil"
)

// dialAMQP opens an AMQP connection and channel from the given URL.
// Returns connection, channel, and a combined closer function.
func dialAMQP(t *testing.T, amqpURL string) (*amqp.Connection, *amqp.Channel) {
	t.Helper()

	conn, err := amqp.Dial(amqpURL)
	require.NoError(t, err, "amqp.Dial failed")
	t.Cleanup(func() { conn.Close() })

	ch, err := conn.Channel()
	require.NoError(t, err, "conn.Channel failed")
	t.Cleanup(func() { ch.Close() })

	return conn, ch
}

// TestDeclareAll_TopologyComplete verifies that DeclareAll provisions the full RabbitMQ
// topology: game.events exchange, game.dlx exchange, all 5 domain queues, and game.dead.queue.
// Uses passive declares to assert existence without modifying topology.
func TestDeclareAll_TopologyComplete(t *testing.T) {
	ctx := context.Background()
	amqpURL, cleanup := testutil.SetupRabbitMQ(ctx, t)
	defer cleanup()

	conn, ch := dialAMQP(t, amqpURL)
	require.NoError(t, rabbitmq.DeclareAll(ch), "DeclareAll failed")

	// Verify main topic exchange.
	err := ch.ExchangeDeclarePassive(rabbitmq.ExchangeGameEvents, amqp.ExchangeTopic,
		true, false, false, false, nil)
	require.NoError(t, err, "passive declare game.events exchange failed")

	// Verify DLX exchange.
	err = ch.ExchangeDeclarePassive(rabbitmq.ExchangeDLX, amqp.ExchangeDirect,
		true, false, false, false, nil)
	require.NoError(t, err, "passive declare game.dlx exchange failed")

	// Verify dead letter queue.
	pCh, err := conn.Channel()
	require.NoError(t, err, "channel for dead queue passive declare failed")
	defer pCh.Close()

	q, err := pCh.QueueDeclarePassive(rabbitmq.QueueDead, true, false, false, false, nil)
	require.NoError(t, err, "game.dead.queue does not exist")
	require.Equal(t, rabbitmq.QueueDead, q.Name)

	// Verify all 5 domain queues.
	queues := []string{
		"ranking.queue",
		"wallet.queue",
		"mission.queue",
		"analytics.queue",
		"notification.queue",
	}
	for _, qName := range queues {
		qCh, err := conn.Channel()
		require.NoError(t, err, "channel for queue %q passive declare failed", qName)

		result, err := qCh.QueueDeclarePassive(qName, true, false, false, false, nil)
		qCh.Close()
		require.NoError(t, err, "queue %q does not exist", qName)
		require.Equal(t, qName, result.Name)
	}
}

// TestDeclareAll_ExchangeExists verifies the game.events exchange is provisioned.
func TestDeclareAll_ExchangeExists(t *testing.T) {
	ctx := context.Background()
	amqpURL, cleanup := testutil.SetupRabbitMQ(ctx, t)
	defer cleanup()

	_, ch := dialAMQP(t, amqpURL)
	require.NoError(t, rabbitmq.DeclareAll(ch), "DeclareAll failed")

	err := ch.ExchangeDeclarePassive(rabbitmq.ExchangeGameEvents, amqp.ExchangeTopic,
		true, false, false, false, nil)
	require.NoError(t, err, "passive declare game.events exchange failed")
}

// TestDeclareAll_AllQueuesExist verifies all 5 domain queues exist after DeclareAll.
func TestDeclareAll_AllQueuesExist(t *testing.T) {
	ctx := context.Background()
	amqpURL, cleanup := testutil.SetupRabbitMQ(ctx, t)
	defer cleanup()

	conn, ch := dialAMQP(t, amqpURL)
	require.NoError(t, rabbitmq.DeclareAll(ch), "DeclareAll failed")

	queues := []string{
		"ranking.queue",
		"wallet.queue",
		"mission.queue",
		"analytics.queue",
		"notification.queue",
	}

	for _, q := range queues {
		pCh, err := conn.Channel()
		require.NoError(t, err, "channel for passive declare failed")
		_, err = pCh.QueueDeclarePassive(q, true, false, false, false, nil)
		pCh.Close()
		require.NoError(t, err, "queue %q does not exist", q)
	}
}

// TestDeclareAll_DLXExists verifies the dead letter exchange and dead letter queue exist.
func TestDeclareAll_DLXExists(t *testing.T) {
	ctx := context.Background()
	amqpURL, cleanup := testutil.SetupRabbitMQ(ctx, t)
	defer cleanup()

	conn, ch := dialAMQP(t, amqpURL)
	require.NoError(t, rabbitmq.DeclareAll(ch), "DeclareAll failed")

	err := ch.ExchangeDeclarePassive(rabbitmq.ExchangeDLX, amqp.ExchangeDirect,
		true, false, false, false, nil)
	require.NoError(t, err, "passive declare game.dlx exchange failed")

	pCh, err := conn.Channel()
	require.NoError(t, err, "channel for passive declare failed")
	defer pCh.Close()

	_, err = pCh.QueueDeclarePassive(rabbitmq.QueueDead, true, false, false, false, nil)
	require.NoError(t, err, "game.dead.queue does not exist")
}

// TestDeclareAll_Idempotent verifies DeclareAll can be called twice on the same channel
// and returns nil both times.
func TestDeclareAll_Idempotent(t *testing.T) {
	ctx := context.Background()
	amqpURL, cleanup := testutil.SetupRabbitMQ(ctx, t)
	defer cleanup()

	_, ch := dialAMQP(t, amqpURL)

	require.NoError(t, rabbitmq.DeclareAll(ch), "DeclareAll first call failed")
	require.NoError(t, rabbitmq.DeclareAll(ch), "DeclareAll second call (idempotent) failed")
}

// TestPublisher_Publish verifies that a message can be published without error.
func TestPublisher_Publish(t *testing.T) {
	ctx := context.Background()
	amqpURL, cleanup := testutil.SetupRabbitMQ(ctx, t)
	defer cleanup()

	logger := zerolog.Nop()
	conn, err := rabbitmq.New(amqpURL, logger)
	require.NoError(t, err, "rabbitmq.New failed")
	defer conn.Close()

	pub, err := rabbitmq.NewPublisher(conn)
	require.NoError(t, err, "NewPublisher failed")
	defer pub.Close()

	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	body := []byte(`{"event":"match.finished","matchId":"test-123"}`)
	require.NoError(t, pub.Publish(ctx2, "match.finished", body), "Publish failed")
}

// TestConsumer_ReceivesMessage verifies end-to-end message delivery: publish then consume.
func TestConsumer_ReceivesMessage(t *testing.T) {
	ctx := context.Background()
	amqpURL, cleanup := testutil.SetupRabbitMQ(ctx, t)
	defer cleanup()

	logger := zerolog.Nop()
	conn, err := rabbitmq.New(amqpURL, logger)
	require.NoError(t, err, "rabbitmq.New failed")
	defer conn.Close()

	ch, err := conn.Channel()
	require.NoError(t, err, "channel failed")
	require.NoError(t, rabbitmq.DeclareAll(ch), "DeclareAll failed")
	ch.Close()

	pub, err := rabbitmq.NewPublisher(conn)
	require.NoError(t, err, "NewPublisher failed")
	defer pub.Close()

	consumer, err := rabbitmq.NewConsumer(conn, "ranking.queue")
	require.NoError(t, err, "NewConsumer failed")

	received := make(chan struct{}, 1)
	ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	go func() {
		_ = consumer.Consume(ctx2, func(d amqp.Delivery) error {
			received <- struct{}{}
			return nil
		})
	}()

	time.Sleep(100 * time.Millisecond)

	body := []byte(`{"event":"match.finished"}`)
	require.NoError(t, pub.Publish(ctx2, "match.finished", body), "Publish failed")

	select {
	case <-received:
		// success
	case <-ctx2.Done():
		t.Error("timed out waiting for message delivery")
	}
}

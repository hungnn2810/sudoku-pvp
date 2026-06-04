//go:build integration

package rabbitmq_test

import (
	"context"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
	"github.com/testcontainers/testcontainers-go"
	tcrabbit "github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

func setupRabbitMQ(t *testing.T) (string, func()) {
	t.Helper()
	ctx := context.Background()

	ctr, err := tcrabbit.Run(ctx, "rabbitmq:3-management-alpine",
		tcrabbit.WithAdminUsername("guest"),
		tcrabbit.WithAdminPassword("guest"),
	)
	if err != nil {
		t.Skipf("Docker not available or RabbitMQ container failed to start: %v", err)
	}

	amqpURL, err := ctr.AmqpURL(ctx)
	if err != nil {
		testcontainers.TerminateContainer(ctr)
		t.Fatalf("get amqp URL: %v", err)
	}

	return amqpURL, func() {
		testcontainers.TerminateContainer(ctr)
	}
}

func TestDeclareAll_ExchangeExists(t *testing.T) {
	amqpURL, cleanup := setupRabbitMQ(t)
	defer cleanup()

	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	defer ch.Close()

	if err := DeclareAll(ch); err != nil {
		t.Fatalf("DeclareAll: %v", err)
	}

	// Passive declare verifies exchange exists without modifying it.
	if err := ch.ExchangeDeclarePassive(ExchangeGameEvents, amqp.ExchangeTopic,
		true, false, false, false, nil); err != nil {
		t.Errorf("passive declare game.events: %v", err)
	}
}

func TestDeclareAll_AllQueuesExist(t *testing.T) {
	amqpURL, cleanup := setupRabbitMQ(t)
	defer cleanup()

	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	defer ch.Close()

	if err := DeclareAll(ch); err != nil {
		t.Fatalf("DeclareAll: %v", err)
	}

	queues := []string{
		"ranking.queue",
		"wallet.queue",
		"mission.queue",
		"analytics.queue",
		"notification.queue",
	}

	// Use a fresh channel for passive declares — passive declare on a channel after error closes it.
	for _, q := range queues {
		pCh, err := conn.Channel()
		if err != nil {
			t.Fatalf("channel for passive declare: %v", err)
		}
		_, err = pCh.QueueDeclarePassive(q, true, false, false, false, nil)
		pCh.Close()
		if err != nil {
			t.Errorf("queue %q does not exist: %v", q, err)
		}
	}
}

func TestDeclareAll_DLXExists(t *testing.T) {
	amqpURL, cleanup := setupRabbitMQ(t)
	defer cleanup()

	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	defer ch.Close()

	if err := DeclareAll(ch); err != nil {
		t.Fatalf("DeclareAll: %v", err)
	}

	// Verify DLX exchange.
	if err := ch.ExchangeDeclarePassive(ExchangeDLX, amqp.ExchangeDirect,
		true, false, false, false, nil); err != nil {
		t.Errorf("passive declare game.dlx: %v", err)
	}

	// Verify dead letter queue.
	pCh, err := conn.Channel()
	if err != nil {
		t.Fatalf("channel for passive declare: %v", err)
	}
	defer pCh.Close()
	_, err = pCh.QueueDeclarePassive(QueueDead, true, false, false, false, nil)
	if err != nil {
		t.Errorf("game.dead.queue does not exist: %v", err)
	}
}

func TestDeclareAll_Idempotent(t *testing.T) {
	amqpURL, cleanup := setupRabbitMQ(t)
	defer cleanup()

	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("channel: %v", err)
	}
	defer ch.Close()

	// Call DeclareAll twice on the same channel — must return nil both times.
	if err := DeclareAll(ch); err != nil {
		t.Fatalf("DeclareAll first call: %v", err)
	}
	if err := DeclareAll(ch); err != nil {
		t.Errorf("DeclareAll second call (idempotent): %v", err)
	}
}

func TestPublisher_Publish(t *testing.T) {
	amqpURL, cleanup := setupRabbitMQ(t)
	defer cleanup()

	logger := zerolog.Nop()
	conn, err := New(amqpURL, logger)
	if err != nil {
		t.Fatalf("New connection: %v", err)
	}
	defer conn.Close()

	pub, err := NewPublisher(conn)
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}
	defer pub.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body := []byte(`{"event":"match.finished","matchId":"test-123"}`)
	if err := pub.Publish(ctx, "match.finished", body); err != nil {
		t.Errorf("Publish: %v", err)
	}
}

func TestConsumer_ReceivesMessage(t *testing.T) {
	amqpURL, cleanup := setupRabbitMQ(t)
	defer cleanup()

	logger := zerolog.Nop()
	conn, err := New(amqpURL, logger)
	if err != nil {
		t.Fatalf("New connection: %v", err)
	}
	defer conn.Close()

	// Declare topology so queues and bindings exist.
	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("setup channel: %v", err)
	}
	if err := DeclareAll(ch); err != nil {
		ch.Close()
		t.Fatalf("DeclareAll: %v", err)
	}
	ch.Close()

	pub, err := NewPublisher(conn)
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}
	defer pub.Close()

	consumer, err := NewConsumer(conn, "ranking.queue")
	if err != nil {
		t.Fatalf("NewConsumer: %v", err)
	}

	received := make(chan struct{}, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	go func() {
		_ = consumer.Consume(ctx, func(d amqp.Delivery) error {
			received <- struct{}{}
			return nil
		})
	}()

	// Give consumer time to start.
	time.Sleep(100 * time.Millisecond)

	body := []byte(`{"event":"match.finished"}`)
	if err := pub.Publish(ctx, "match.finished", body); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case <-received:
		// success
	case <-ctx.Done():
		t.Error("timed out waiting for message delivery")
	}
}

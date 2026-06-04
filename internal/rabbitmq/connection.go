package rabbitmq

import (
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

const reconnectDelay = 5 * time.Second

// Connection wraps an AMQP connection with automatic reconnection on failure.
// It is goroutine-safe: the internal connection is protected by a read/write mutex.
// Each caller to Channel() receives its own AMQP channel — channels must never be
// shared across goroutines (amqp091-go channels are not goroutine-safe).
type Connection struct {
	mu     sync.RWMutex
	conn   *amqp.Connection
	url    string
	logger zerolog.Logger
	closed bool
	done   chan struct{} // closed by Close() to interrupt the reconnect loop immediately
}

// New dials the AMQP broker synchronously (fail-fast at startup), declares topology,
// then launches a background reconnect goroutine that re-dials and re-declares
// topology whenever the connection drops.
//
// The AMQP URL must NOT be logged — it contains credentials (T-03-01).
func New(url string, logger zerolog.Logger) (*Connection, error) {
	c := &Connection{url: url, logger: logger, done: make(chan struct{})}

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}
	c.setConn(conn)

	// Declare topology on the initial connection.
	if err := c.declareTopology(conn); err != nil {
		return nil, fmt.Errorf("rabbitmq initial topology: %w", err)
	}

	go c.reconnectLoop()

	return c, nil
}

// Channel opens a new AMQP channel from the current connection.
// Each caller gets its own channel — never share channels across goroutines.
func (c *Connection) Channel() (*amqp.Channel, error) {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()

	if conn == nil || conn.IsClosed() {
		return nil, fmt.Errorf("rabbitmq: connection is not open")
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("rabbitmq open channel: %w", err)
	}
	return ch, nil
}

// Close closes the underlying AMQP connection and stops the reconnect loop.
// The done channel is closed to interrupt any in-progress reconnect delay immediately.
func (c *Connection) Close() {
	c.mu.Lock()
	c.closed = true
	conn := c.conn
	c.mu.Unlock()

	close(c.done)

	if conn != nil && !conn.IsClosed() {
		conn.Close()
	}
}

// setConn safely replaces the current connection.
func (c *Connection) setConn(conn *amqp.Connection) {
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
}

// declareTopology opens a short-lived channel, calls DeclareAll, and closes the channel.
func (c *Connection) declareTopology(conn *amqp.Connection) error {
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open topology channel: %w", err)
	}
	defer ch.Close()
	return DeclareAll(ch)
}

// reconnectLoop watches the NotifyClose channel and re-dials on connection loss.
// It retries indefinitely with a fixed delay between attempts.
// Credentials are never logged — only connection events are emitted.
//
// CR-04 fix: after a successful reconnect, the local conn variable is updated to
// newConn so that the outer loop's next NotifyClose call is registered on the
// live connection, not the stale closed one.
//
// WR-04 fix: the inter-retry delay uses a select on time.After + c.done so that
// a Close() call interrupts the sleep immediately rather than blocking up to 5s.
func (c *Connection) reconnectLoop() {
	c.mu.RLock()
	conn := c.conn
	isClosed := c.closed
	c.mu.RUnlock()

	if isClosed {
		return
	}

	for {
		// Register for close notifications on the current connection.
		notify := conn.NotifyClose(make(chan *amqp.Error, 1))
		<-notify // block until connection drops

		c.mu.RLock()
		isClosed = c.closed
		c.mu.RUnlock()
		if isClosed {
			return
		}

		c.logger.Warn().Msg("rabbitmq connection closed, reconnecting")

		// Retry until a successful dial.
		for {
			c.mu.RLock()
			isClosed = c.closed
			c.mu.RUnlock()
			if isClosed {
				return
			}

			newConn, err := amqp.Dial(c.url)
			if err != nil {
				// Log only the fact that dial failed, never the URL (T-03-01).
				c.logger.Error().Err(err).Msg("rabbitmq dial failed, retrying in 5s")
				// Interruptible delay: Close() signals done immediately.
				select {
				case <-time.After(reconnectDelay):
				case <-c.done:
					return
				}
				continue
			}

			c.setConn(newConn)
			c.logger.Info().Msg("rabbitmq reconnected successfully")

			if err := c.declareTopology(newConn); err != nil {
				c.logger.Error().Err(err).Msg("rabbitmq topology redeclaration failed after reconnect")
			}

			// Update local reference so next iteration's NotifyClose targets the
			// live connection, not the stale one (CR-04).
			conn = newConn
			break
		}
	}
}

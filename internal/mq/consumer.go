package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitMQConsumer implements Consumer interface
type RabbitMQConsumer struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	queues     []string
	prefetch   int
	consumerID string
}

// ConsumerConfig holds consumer configuration
type ConsumerConfig struct {
	URL        string
	Prefetch   int    // Number of messages to prefetch (default: 10)
	ConsumerID string // Unique consumer identifier
	Queues     []string // Queues to consume from (default: all priority queues)
}

// NewConsumer creates a new RabbitMQ consumer
func NewConsumer(cfg ConsumerConfig) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial failed: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("rabbitmq channel failed: %w", err)
	}

	// Set prefetch
	prefetch := cfg.Prefetch
	if prefetch <= 0 {
		prefetch = 10
	}
	if err := ch.Qos(prefetch, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("set qos failed: %w", err)
	}

	// Default queues: consume from all priority queues
	queues := cfg.Queues
	if len(queues) == 0 {
		queues = []string{QueueCritical, QueueHigh, QueueDefault, QueueLow}
	}

	// Ensure queues exist
	for _, qName := range queues {
		if _, err := ch.QueueDeclare(
			qName,
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			nil,   // arguments
		); err != nil {
			ch.Close()
			conn.Close()
			return nil, fmt.Errorf("queue declare %s failed: %w", qName, err)
		}
	}

	consumerID := cfg.ConsumerID
	if consumerID == "" {
		consumerID = fmt.Sprintf("worker-%d", time.Now().UnixNano())
	}

	return &RabbitMQConsumer{
		conn:       conn,
		channel:    ch,
		queues:     queues,
		prefetch:   prefetch,
		consumerID: consumerID,
	}, nil
}

// Consume starts consuming messages from all configured queues
func (c *RabbitMQConsumer) Consume(ctx context.Context, handler func(context.Context, *JobMessage) error) error {
	// Create channels for each queue
	var deliveryChannels []<-chan amqp.Delivery

	for _, qName := range c.queues {
		deliveries, err := c.channel.Consume(
			qName,
			fmt.Sprintf("%s-%s", c.consumerID, qName),
			false, // auto-ack (we manually ack)
			false, // exclusive
			false, // no-local
			false, // no-wait
			nil,   // args
		)
		if err != nil {
			return fmt.Errorf("consume from %s failed: %w", qName, err)
		}
		deliveryChannels = append(deliveryChannels, deliveries)
	}

	// Merge all delivery channels with context for clean shutdown
	merged := mergeChannelsWithContext(ctx, deliveryChannels...)

	// Retry backoff configuration
	const (
		initialBackoff = 1 * time.Second
		maxBackoff     = 30 * time.Second
		maxRetries     = 5
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case d, ok := <-merged:
			if !ok {
				return fmt.Errorf("delivery channel closed")
			}

			var msg JobMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Printf("[Consumer] Failed to unmarshal message: %v", err)
				// Reject and don't requeue malformed messages
				d.Reject(false)
				continue
			}

			// Track retry count from message header
			retryCount := 0
			if d.Headers != nil {
				if count, ok := d.Headers["x-retry-count"].(int64); ok {
					retryCount = int(count)
				} else if count, ok := d.Headers["x-retry-count"].(int32); ok {
					retryCount = int(count)
				}
			}

			// Process the message
			if err := handler(ctx, &msg); err != nil {
				log.Printf("[Consumer] Handler failed for job %s (retry %d/%d): %v", msg.JobID, retryCount, maxRetries, err)

				if retryCount >= maxRetries {
					// Max retries exceeded, dead-letter the message
					log.Printf("[Consumer] Max retries exceeded for job %s, rejecting without requeue", msg.JobID)
					d.Reject(false)
					continue
				}

				// Calculate backoff with exponential increase
				backoff := initialBackoff * time.Duration(1<<uint(retryCount))
				if backoff > maxBackoff {
					backoff = maxBackoff
				}

				log.Printf("[Consumer] Waiting %v before republishing job %s (retry %d)", backoff, msg.JobID, retryCount+1)

				// Wait before republishing (prevents tight retry loop)
				select {
				case <-ctx.Done():
					// Context cancelled during backoff, reject without requeue
					d.Reject(false)
					return ctx.Err()
				case <-time.After(backoff):
					// Republish with incremented retry count header
					// Native requeue doesn't preserve headers, so we republish manually
					headers := amqp.Table{
						"x-retry-count": int64(retryCount + 1),
					}

					err := c.channel.PublishWithContext(ctx,
						"",           // exchange (use default)
						d.RoutingKey, // routing key = queue name
						false,        // mandatory
						false,        // immediate
						amqp.Publishing{
							ContentType:  "application/json",
							DeliveryMode: amqp.Persistent,
							Headers:      headers,
							Body:         d.Body,
						},
					)
					if err != nil {
						log.Printf("[Consumer] Failed to republish job %s: %v, rejecting without requeue", msg.JobID, err)
						d.Reject(false)
					} else {
						// Acknowledge original message since we've republished
						d.Ack(false)
					}
				}
				continue
			}

			// Acknowledge successful processing
			if err := d.Ack(false); err != nil {
				log.Printf("[Consumer] Ack failed for job %s: %v", msg.JobID, err)
			}
		}
	}
}

// Close closes the consumer connection
func (c *RabbitMQConsumer) Close() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// mergeChannelsWithContext merges multiple channels into one with proper cleanup
// Uses context to prevent goroutine leaks on shutdown
func mergeChannelsWithContext(ctx context.Context, channels ...<-chan amqp.Delivery) <-chan amqp.Delivery {
	merged := make(chan amqp.Delivery)
	var wg sync.WaitGroup

	for _, ch := range channels {
		wg.Add(1)
		go func(c <-chan amqp.Delivery) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					// Context cancelled, drain remaining messages without blocking
					return
				case d, ok := <-c:
					if !ok {
						return
					}
					select {
					case merged <- d:
					case <-ctx.Done():
						// Context cancelled while trying to send
						return
					}
				}
			}
		}(ch)
	}

	// Close merged channel when all source channels are closed or context is done
	go func() {
		wg.Wait()
		close(merged)
	}()

	return merged
}

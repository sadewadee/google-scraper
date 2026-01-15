package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

// JobMessage represents a job message in the queue
type JobMessage struct {
	JobID    uuid.UUID `json:"job_id"`
	Priority int       `json:"priority"`
	Type     string    `json:"type"` // job:process
}

// Config holds RabbitMQ connection configuration
type Config struct {
	URL string // amqp://user:pass@host:5672/vhost
}

// Queue names and routing keys
const (
	ExchangeName = "gmaps.jobs"

	QueueCritical = "gmaps.jobs.critical"
	QueueHigh     = "gmaps.jobs.high"
	QueueDefault  = "gmaps.jobs.default"
	QueueLow      = "gmaps.jobs.low"

	RoutingKeyCritical = "critical"
	RoutingKeyHigh     = "high"
	RoutingKeyDefault  = "default"
	RoutingKeyLow      = "low"
)

// PriorityToRoutingKey converts a priority value to a routing key
func PriorityToRoutingKey(priority int) string {
	switch {
	case priority >= 10:
		return RoutingKeyCritical
	case priority >= 5:
		return RoutingKeyHigh
	case priority < 0:
		return RoutingKeyLow
	default:
		return RoutingKeyDefault
	}
}

// Publisher interface for publishing messages to RabbitMQ
type Publisher interface {
	Publish(ctx context.Context, msg *JobMessage) error
	Close() error
}

// Consumer interface for consuming messages from RabbitMQ
type Consumer interface {
	Consume(ctx context.Context, handler func(context.Context, *JobMessage) error) error
	Close() error
}

// RabbitMQPublisher implements Publisher interface
type RabbitMQPublisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewPublisher creates a new RabbitMQ publisher
func NewPublisher(cfg Config) (*RabbitMQPublisher, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial failed: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("rabbitmq channel failed: %w", err)
	}

	// Declare exchange
	if err := ch.ExchangeDeclare(
		ExchangeName,
		"direct",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("exchange declare failed: %w", err)
	}

	// Declare queues
	queues := []struct {
		name       string
		routingKey string
	}{
		{QueueCritical, RoutingKeyCritical},
		{QueueHigh, RoutingKeyHigh},
		{QueueDefault, RoutingKeyDefault},
		{QueueLow, RoutingKeyLow},
	}

	for _, q := range queues {
		if _, err := ch.QueueDeclare(
			q.name,
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			nil,   // arguments
		); err != nil {
			ch.Close()
			conn.Close()
			return nil, fmt.Errorf("queue declare %s failed: %w", q.name, err)
		}

		if err := ch.QueueBind(
			q.name,
			q.routingKey,
			ExchangeName,
			false,
			nil,
		); err != nil {
			ch.Close()
			conn.Close()
			return nil, fmt.Errorf("queue bind %s failed: %w", q.name, err)
		}
	}

	return &RabbitMQPublisher{
		conn:    conn,
		channel: ch,
	}, nil
}

// Publish publishes a job message to RabbitMQ
func (p *RabbitMQPublisher) Publish(ctx context.Context, msg *JobMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message failed: %w", err)
	}

	routingKey := PriorityToRoutingKey(msg.Priority)

	return p.channel.PublishWithContext(
		ctx,
		ExchangeName,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
}

// Close closes the publisher connection
func (p *RabbitMQPublisher) Close() error {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}

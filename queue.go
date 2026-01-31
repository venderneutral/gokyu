// Package goqueue provides a cloud-agnostic message queue abstraction
// using AMQP 1.0 protocol for communication with various cloud providers.
package gokyu

import (
	"context"
)

// Provider represents a supported queue provider.
type Provider string

const (
	ProviderAzure    Provider = "azure"
	ProviderAmazonMQ Provider = "amazonmq"
)

// Message represents a queue message with provider-agnostic fields.
type Message struct {
	// ID is the unique identifier of the message (if provided by the broker).
	ID string

	// Body is the message payload.
	Body []byte

	// Properties contains optional message properties/headers.
	Properties map[string]interface{}

	// raw holds the provider-specific message for acknowledgment operations.
	raw interface{}
}

// NewMessage creates a new message with the given body.
func NewMessage(body []byte) *Message {
	return &Message{
		Body:       body,
		Properties: make(map[string]interface{}),
	}
}

// Raw returns the provider-specific raw message (used for acknowledgment).
func (m *Message) Raw() interface{} {
	return m.raw
}

// SetRaw sets the provider-specific raw message.
func (m *Message) SetRaw(raw interface{}) {
	m.raw = raw
}

// Publisher defines the interface for publishing messages to a queue or topic.
type Publisher interface {
	// Publish sends a message to the configured destination.
	Publish(ctx context.Context, msg *Message) error

	// Close releases resources associated with the publisher.
	Close(ctx context.Context) error
}

// Subscriber defines the interface for receiving messages from a queue or subscription.
type Subscriber interface {
	// Receive blocks until a message is available or the context is cancelled.
	Receive(ctx context.Context) (*Message, error)

	// Ack acknowledges successful processing of a message.
	Ack(ctx context.Context, msg *Message) error

	// Nack negatively acknowledges a message (typically for redelivery or dead-lettering).
	Nack(ctx context.Context, msg *Message) error

	// Close releases resources associated with the subscriber.
	Close(ctx context.Context) error
}

// ProviderFactory creates publishers and subscribers for a specific provider.
type ProviderFactory interface {
	// NewPublisher creates a new publisher for the given configuration.
	NewPublisher(ctx context.Context, cfg *Config) (Publisher, error)

	// NewSubscriber creates a new subscriber for the given configuration.
	NewSubscriber(ctx context.Context, cfg *Config) (Subscriber, error)
}

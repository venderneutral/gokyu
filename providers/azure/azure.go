// Package azure provides Azure Service Bus implementation for gokyu.
//
// This package implements the gokyu.Publisher and gokyu.Subscriber interfaces
// for Azure Service Bus using AMQP 1.0 protocol.
//
// # Connection String Format
//
// Azure Service Bus connection strings should follow this format:
//
//	amqps://<policy-name>:<access-key>@<namespace>.servicebus.windows.net
//
// # Topic Subscriptions
//
// For pub/sub messaging, Azure Service Bus uses the following addressing:
//   - Topic: "my-topic"
//   - Subscription: "my-topic/Subscriptions/my-subscription"
//
// The subscription path is automatically constructed by this package
// when you provide Topic and Subscription in the configuration.
//
// # Usage
//
// Import this package to register the Azure provider:
//
//	import _ "github.com/venderneutral/gokyu/providers/azure"
package azure

import (
	"context"
	"fmt"

	"github.com/Azure/go-amqp"
	"github.com/venderneutral/gokyu"
)

func init() {
	gokyu.RegisterProvider(gokyu.ProviderAzure, &Factory{})
}

// Factory creates Azure Service Bus publishers and subscribers.
type Factory struct{}

// NewPublisher creates a new Azure Service Bus publisher.
func (f *Factory) NewPublisher(ctx context.Context, cfg *gokyu.Config) (gokyu.Publisher, error) {
	conn, err := amqp.Dial(ctx, cfg.BuildConnectionString(), nil)
	if err != nil {
		return nil, gokyu.WrapError(gokyu.ErrConnectionFailed, err)
	}

	session, err := conn.NewSession(ctx, nil)
	if err != nil {
		conn.Close()
		return nil, gokyu.WrapError(gokyu.ErrConnectionFailed, err)
	}

	// Determine destination (topic or queue)
	destination := cfg.Topic
	if destination == "" {
		destination = cfg.Queue
	}

	sender, err := session.NewSender(ctx, destination, nil)
	if err != nil {
		session.Close(ctx)
		conn.Close()
		return nil, gokyu.WrapError(gokyu.ErrConnectionFailed, err)
	}

	return &publisher{
		conn:    conn,
		session: session,
		sender:  sender,
	}, nil
}

// NewSubscriber creates a new Azure Service Bus subscriber.
func (f *Factory) NewSubscriber(ctx context.Context, cfg *gokyu.Config) (gokyu.Subscriber, error) {
	conn, err := amqp.Dial(ctx, cfg.BuildConnectionString(), nil)
	if err != nil {
		return nil, gokyu.WrapError(gokyu.ErrConnectionFailed, err)
	}

	session, err := conn.NewSession(ctx, nil)
	if err != nil {
		conn.Close()
		return nil, gokyu.WrapError(gokyu.ErrConnectionFailed, err)
	}

	// Build the source address
	source := buildSourceAddress(cfg)

	receiver, err := session.NewReceiver(ctx, source, nil)
	if err != nil {
		session.Close(ctx)
		conn.Close()
		return nil, gokyu.WrapError(gokyu.ErrConnectionFailed, err)
	}

	return &subscriber{
		conn:     conn,
		session:  session,
		receiver: receiver,
	}, nil
}

// buildSourceAddress constructs the AMQP source address for Azure Service Bus.
func buildSourceAddress(cfg *gokyu.Config) string {
	if cfg.Queue != "" {
		return cfg.Queue
	}
	// For topics, Azure uses: topic/subscriptions/subscription-name
	return fmt.Sprintf("%s/Subscriptions/%s", cfg.Topic, cfg.Subscription)
}

// publisher implements gokyu.Publisher for Azure Service Bus.
type publisher struct {
	conn    *amqp.Conn
	session *amqp.Session
	sender  *amqp.Sender
}

func (p *publisher) Publish(ctx context.Context, msg *gokyu.Message) error {
	amqpMsg := amqp.NewMessage(msg.Body)

	// Set message ID if provided
	if msg.ID != "" {
		amqpMsg.Properties = &amqp.MessageProperties{
			MessageID: msg.ID,
		}
	}

	// Set application properties
	if len(msg.Properties) > 0 {
		amqpMsg.ApplicationProperties = msg.Properties
	}

	if err := p.sender.Send(ctx, amqpMsg, nil); err != nil {
		return gokyu.WrapError(gokyu.ErrPublishFailed, err)
	}
	return nil
}

func (p *publisher) Close(ctx context.Context) error {
	var errs []error

	if err := p.sender.Close(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := p.session.Close(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := p.conn.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// subscriber implements gokyu.Subscriber for Azure Service Bus.
type subscriber struct {
	conn     *amqp.Conn
	session  *amqp.Session
	receiver *amqp.Receiver
}

func (s *subscriber) Receive(ctx context.Context) (*gokyu.Message, error) {
	amqpMsg, err := s.receiver.Receive(ctx, nil)
	if err != nil {
		return nil, gokyu.WrapError(gokyu.ErrReceiveFailed, err)
	}

	msg := &gokyu.Message{
		Body:       amqpMsg.GetData(),
		Properties: make(map[string]interface{}),
	}

	// Extract message ID
	if amqpMsg.Properties != nil && amqpMsg.Properties.MessageID != nil {
		msg.ID = fmt.Sprintf("%v", amqpMsg.Properties.MessageID)
	}

	// Extract application properties
	for k, v := range amqpMsg.ApplicationProperties {
		msg.Properties[k] = v
	}

	// Store raw message for acknowledgment
	msg.SetRaw(amqpMsg)

	return msg, nil
}

func (s *subscriber) Ack(ctx context.Context, msg *gokyu.Message) error {
	amqpMsg, ok := msg.Raw().(*amqp.Message)
	if !ok {
		return gokyu.ErrAckFailed
	}
	if err := s.receiver.AcceptMessage(ctx, amqpMsg); err != nil {
		return gokyu.WrapError(gokyu.ErrAckFailed, err)
	}
	return nil
}

func (s *subscriber) Nack(ctx context.Context, msg *gokyu.Message) error {
	amqpMsg, ok := msg.Raw().(*amqp.Message)
	if !ok {
		return gokyu.ErrAckFailed
	}
	// Release the message for redelivery
	if err := s.receiver.ReleaseMessage(ctx, amqpMsg); err != nil {
		return gokyu.WrapError(gokyu.ErrAckFailed, err)
	}
	return nil
}

func (s *subscriber) Close(ctx context.Context) error {
	var errs []error

	if err := s.receiver.Close(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := s.session.Close(ctx); err != nil {
		errs = append(errs, err)
	}
	if err := s.conn.Close(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

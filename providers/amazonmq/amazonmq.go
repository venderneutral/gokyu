// Package amazonmq provides Amazon MQ (ActiveMQ) implementation for gokyu.
package amazonmq

import (
	"context"
	"fmt"

	"github.com/Azure/go-amqp"
	"github.com/venderneutral/gokyu"
)

func init() {
	gokyu.RegisterProvider(gokyu.ProviderAmazonMQ, &Factory{})
}

// Factory creates Amazon MQ publishers and subscribers.
type Factory struct{}

// NewPublisher creates a new Amazon MQ publisher.
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

	// Build destination address for ActiveMQ
	destination := buildDestinationAddress(cfg)

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

// NewSubscriber creates a new Amazon MQ subscriber.
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

	// Build source address for ActiveMQ
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

// buildDestinationAddress constructs the AMQP address for Amazon MQ (ActiveMQ).
// ActiveMQ uses JMS-style addressing: queue://name or topic://name
func buildDestinationAddress(cfg *gokyu.Config) string {
	if cfg.Queue != "" {
		return cfg.Queue
	}
	return fmt.Sprintf("topic://%s", cfg.Topic)
}

// buildSourceAddress constructs the AMQP source address for Amazon MQ.
// For topics with durable subscriptions, ActiveMQ uses: Consumer.<subscription-name>.VirtualTopic.<topic-name>
func buildSourceAddress(cfg *gokyu.Config) string {
	if cfg.Queue != "" {
		return cfg.Queue
	}
	// ActiveMQ Virtual Topics pattern for durable subscriptions
	// Consumer.<client-id>.<subscription>.VirtualTopic.<topic-name>
	if cfg.Subscription != "" {
		return fmt.Sprintf("Consumer.%s.VirtualTopic.%s", cfg.Subscription, cfg.Topic)
	}
	return fmt.Sprintf("topic://%s", cfg.Topic)
}

// publisher implements gokyu.Publisher for Amazon MQ.
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

// subscriber implements gokyu.Subscriber for Amazon MQ.
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

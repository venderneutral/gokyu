package gokyu

import (
	"context"
	"errors"
	"testing"
)

// mockFactory is a test factory for verifying client behavior.
type mockFactory struct {
	publisherErr  error
	subscriberErr error
}

func (f *mockFactory) NewPublisher(ctx context.Context, cfg *Config) (Publisher, error) {
	if f.publisherErr != nil {
		return nil, f.publisherErr
	}
	return &mockPublisher{}, nil
}

func (f *mockFactory) NewSubscriber(ctx context.Context, cfg *Config) (Subscriber, error) {
	if f.subscriberErr != nil {
		return nil, f.subscriberErr
	}
	return &mockSubscriber{}, nil
}

type mockPublisher struct{}

func (p *mockPublisher) Publish(ctx context.Context, msg *Message) error { return nil }
func (p *mockPublisher) Close(ctx context.Context) error                 { return nil }

type mockSubscriber struct{}

func (s *mockSubscriber) Receive(ctx context.Context) (*Message, error) { return nil, nil }
func (s *mockSubscriber) Ack(ctx context.Context, msg *Message) error   { return nil }
func (s *mockSubscriber) Nack(ctx context.Context, msg *Message) error  { return nil }
func (s *mockSubscriber) Close(ctx context.Context) error               { return nil }

func TestNewClient(t *testing.T) {
	// Register a test provider
	testProvider := Provider("test-provider")
	RegisterProvider(testProvider, &mockFactory{})

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Provider:         testProvider,
				ConnectionString: "amqps://test@host",
				Topic:            "topic",
			},
			wantErr: false,
		},
		{
			name: "invalid config",
			config: &Config{
				Provider: testProvider,
				// Missing connection info
			},
			wantErr: true,
		},
		{
			name: "unsupported provider",
			config: &Config{
				Provider:         "unknown-provider",
				ConnectionString: "amqps://test@host",
				Topic:            "topic",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && client == nil {
				t.Error("expected client to be non-nil")
			}
		})
	}
}

func TestClient_NewPublisher(t *testing.T) {
	testProvider := Provider("test-pub-provider")

	t.Run("success", func(t *testing.T) {
		RegisterProvider(testProvider, &mockFactory{})

		client, _ := NewClient(&Config{
			Provider:         testProvider,
			ConnectionString: "amqps://test",
			Topic:            "topic",
		})

		pub, err := client.NewPublisher(context.Background())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if pub == nil {
			t.Error("expected publisher to be non-nil")
		}
	})

	t.Run("factory error", func(t *testing.T) {
		errProvider := Provider("test-pub-err-provider")
		RegisterProvider(errProvider, &mockFactory{
			publisherErr: errors.New("connection refused"),
		})

		client, _ := NewClient(&Config{
			Provider:         errProvider,
			ConnectionString: "amqps://test",
			Topic:            "topic",
		})

		_, err := client.NewPublisher(context.Background())
		if err == nil {
			t.Error("expected error from factory")
		}
	})
}

func TestClient_NewSubscriber(t *testing.T) {
	testProvider := Provider("test-sub-provider")

	t.Run("success", func(t *testing.T) {
		RegisterProvider(testProvider, &mockFactory{})

		client, _ := NewClient(&Config{
			Provider:         testProvider,
			ConnectionString: "amqps://test",
			Topic:            "topic",
			Subscription:     "sub",
		})

		sub, err := client.NewSubscriber(context.Background())
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if sub == nil {
			t.Error("expected subscriber to be non-nil")
		}
	})
}

func TestClient_Config(t *testing.T) {
	testProvider := Provider("test-cfg-provider")
	RegisterProvider(testProvider, &mockFactory{})

	originalCfg := &Config{
		Provider:         testProvider,
		ConnectionString: "amqps://test",
		Topic:            "my-topic",
		Subscription:     "my-sub",
	}

	client, _ := NewClient(originalCfg)
	returnedCfg := client.Config()

	if returnedCfg.Topic != "my-topic" {
		t.Errorf("expected Topic 'my-topic', got '%s'", returnedCfg.Topic)
	}
	if returnedCfg.Subscription != "my-sub" {
		t.Errorf("expected Subscription 'my-sub', got '%s'", returnedCfg.Subscription)
	}
}

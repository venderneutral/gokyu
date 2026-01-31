package gokyu

import (
	"context"
	"sync"
)

// Client provides a unified interface for creating publishers and subscribers.
type Client struct {
	config  *Config
	factory ProviderFactory
}

// registry holds registered provider factories.
var (
	registryMu sync.RWMutex
	registry   = make(map[Provider]ProviderFactory)
)

// RegisterProvider registers a provider factory for the given provider name.
// This is typically called by provider packages in their init() functions.
func RegisterProvider(name Provider, factory ProviderFactory) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[name] = factory
}

// getFactory returns the factory for the given provider.
func getFactory(provider Provider) (ProviderFactory, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	factory, ok := registry[provider]
	if !ok {
		return nil, ErrUnsupportedProvider
	}
	return factory, nil
}

// NewClient creates a new client with the given configuration.
func NewClient(cfg *Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	factory, err := getFactory(cfg.Provider)
	if err != nil {
		return nil, err
	}

	return &Client{
		config:  cfg,
		factory: factory,
	}, nil
}

// NewClientFromEnv creates a new client using environment variables.
func NewClientFromEnv() (*Client, error) {
	cfg, err := LoadConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return NewClient(cfg)
}

// NewPublisher creates a new publisher using the configured provider.
func (c *Client) NewPublisher(ctx context.Context) (Publisher, error) {
	return c.factory.NewPublisher(ctx, c.config)
}

// NewSubscriber creates a new subscriber using the configured provider.
func (c *Client) NewSubscriber(ctx context.Context) (Subscriber, error) {
	return c.factory.NewSubscriber(ctx, c.config)
}

// Config returns a copy of the client's configuration.
func (c *Client) Config() Config {
	return *c.config
}

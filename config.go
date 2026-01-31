package gokyu

import (
	"fmt"
	"net/url"
	"os"
)

// Config holds the configuration for connecting to a message queue.
type Config struct {
	// Provider specifies which cloud provider to use.
	Provider Provider

	// ConnectionString is the full AMQP connection string.
	// If provided, it takes precedence over individual connection parameters.
	ConnectionString string

	// Host is the broker hostname (used if ConnectionString is not provided).
	Host string

	// Port is the broker port (default: 5671 for AMQPS).
	Port int

	// Username for authentication (used if ConnectionString is not provided).
	Username string

	// Password for authentication (used if ConnectionString is not provided).
	Password string

	// UseTLS enables TLS/SSL connection (default: true for cloud providers).
	UseTLS bool

	// Queue is the name of the queue for point-to-point messaging.
	Queue string

	// Topic is the name of the topic for pub/sub messaging.
	Topic string

	// Subscription is the name of the subscription (required for receiving from topics).
	Subscription string
}

// Validate checks that the configuration has all required fields.
func (c *Config) Validate() error {
	if c.Provider == "" {
		return ErrInvalidConfig("provider is required")
	}

	if c.ConnectionString == "" {
		if c.Host == "" {
			return ErrInvalidConfig("host or connection_string is required")
		}
		if c.Username == "" || c.Password == "" {
			return ErrInvalidConfig("username and password are required when connection_string is not provided")
		}
	}

	if c.Queue == "" && c.Topic == "" {
		return ErrInvalidConfig("either queue or topic must be specified")
	}

	return nil
}

// BuildConnectionString constructs an AMQP connection string from individual parameters.
func (c *Config) BuildConnectionString() string {
	if c.ConnectionString != "" {
		return c.ConnectionString
	}

	scheme := "amqps"
	if !c.UseTLS {
		scheme = "amqp"
	}

	port := c.Port
	if port == 0 {
		if c.UseTLS {
			port = 5671
		} else {
			port = 5672
		}
	}

	encodedPassword := url.QueryEscape(c.Password)
	return fmt.Sprintf("%s://%s:%s@%s:%d", scheme, c.Username, encodedPassword, c.Host, port)
}

// Environment variable names for configuration.
const (
	EnvProvider         = "GOKYU_PROVIDER"
	EnvConnectionString = "GOKYU_CONNECTION_STRING"
	EnvHost             = "GOKYU_HOST"
	EnvPort             = "GOKYU_PORT"
	EnvUsername         = "GOKYU_USERNAME"
	EnvPassword         = "GOKYU_PASSWORD"
	EnvQueue            = "GOKYU_QUEUE"
	EnvTopic            = "GOKYU_TOPIC"
	EnvSubscription     = "GOKYU_SUBSCRIPTION"
)

// LoadConfigFromEnv creates a Config from environment variables.
func LoadConfigFromEnv() (*Config, error) {
	cfg := &Config{
		Provider:         Provider(os.Getenv(EnvProvider)),
		ConnectionString: os.Getenv(EnvConnectionString),
		Host:             os.Getenv(EnvHost),
		Username:         os.Getenv(EnvUsername),
		Password:         os.Getenv(EnvPassword),
		Queue:            os.Getenv(EnvQueue),
		Topic:            os.Getenv(EnvTopic),
		Subscription:     os.Getenv(EnvSubscription),
		UseTLS:           true,
	}

	if portStr := os.Getenv(EnvPort); portStr != "" {
		var port int
		if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
			return nil, ErrInvalidConfig("invalid port number")
		}
		cfg.Port = port
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

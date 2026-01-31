package gokyu

import (
	"errors"
	"fmt"
)

// Common sentinel errors.
var (
	// ErrConnectionFailed indicates a connection to the broker could not be established.
	ErrConnectionFailed = errors.New("gokyu: connection failed")

	// ErrPublishFailed indicates a message could not be published.
	ErrPublishFailed = errors.New("gokyu: publish failed")

	// ErrReceiveFailed indicates a message could not be received.
	ErrReceiveFailed = errors.New("gokyu: receive failed")

	// ErrAckFailed indicates a message acknowledgment failed.
	ErrAckFailed = errors.New("gokyu: acknowledgment failed")

	// ErrClosed indicates an operation was attempted on a closed connection.
	ErrClosed = errors.New("gokyu: connection closed")

	// ErrUnsupportedProvider indicates the specified provider is not supported.
	ErrUnsupportedProvider = errors.New("gokyu: unsupported provider")
)

// ConfigError represents a configuration validation error.
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("gokyu: invalid config: %s", e.Message)
}

// ErrInvalidConfig creates a new configuration error.
func ErrInvalidConfig(msg string) error {
	return &ConfigError{Message: msg}
}

// WrapError wraps an error with a sentinel error for easier error checking.
func WrapError(sentinel error, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %v", sentinel, err)
}

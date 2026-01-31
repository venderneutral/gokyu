package gokyu

import (
	"testing"
)

func TestNewMessage(t *testing.T) {
	body := []byte("test message")
	msg := NewMessage(body)

	if string(msg.Body) != "test message" {
		t.Errorf("expected body 'test message', got '%s'", string(msg.Body))
	}

	if msg.Properties == nil {
		t.Error("expected Properties to be initialized")
	}

	if len(msg.Properties) != 0 {
		t.Error("expected empty Properties map")
	}
}

func TestMessage_RawAccessors(t *testing.T) {
	msg := NewMessage([]byte("test"))

	if msg.Raw() != nil {
		t.Error("expected Raw() to be nil initially")
	}

	rawValue := "test-raw"
	msg.SetRaw(rawValue)

	if msg.Raw() != rawValue {
		t.Errorf("expected Raw() to be '%s', got '%v'", rawValue, msg.Raw())
	}
}

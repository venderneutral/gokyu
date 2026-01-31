package gokyu

import (
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid with connection string and topic",
			config: Config{
				Provider:         ProviderAzure,
				ConnectionString: "amqps://test:key@host",
				Topic:            "my-topic",
			},
			wantErr: false,
		},
		{
			name: "valid with connection string and queue",
			config: Config{
				Provider:         ProviderAzure,
				ConnectionString: "amqps://test:key@host",
				Queue:            "my-queue",
			},
			wantErr: false,
		},
		{
			name: "valid with host credentials",
			config: Config{
				Provider: ProviderAmazonMQ,
				Host:     "broker.mq.amazonaws.com",
				Username: "admin",
				Password: "secret",
				Queue:    "my-queue",
			},
			wantErr: false,
		},
		{
			name:    "missing provider",
			config:  Config{ConnectionString: "amqps://test", Topic: "topic"},
			wantErr: true,
		},
		{
			name:    "missing host and connection string",
			config:  Config{Provider: ProviderAzure, Topic: "topic"},
			wantErr: true,
		},
		{
			name: "missing credentials without connection string",
			config: Config{
				Provider: ProviderAzure,
				Host:     "host.com",
				Topic:    "topic",
			},
			wantErr: true,
		},
		{
			name: "missing queue and topic",
			config: Config{
				Provider:         ProviderAzure,
				ConnectionString: "amqps://test:key@host",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_BuildConnectionString(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   string
	}{
		{
			name: "returns existing connection string",
			config: Config{
				ConnectionString: "amqps://existing@host",
				Host:             "ignored.com",
				Username:         "ignored",
				Password:         "ignored",
			},
			want: "amqps://existing@host",
		},
		{
			name: "builds with TLS",
			config: Config{
				Host:     "broker.com",
				Username: "user",
				Password: "pass",
				UseTLS:   true,
			},
			want: "amqps://user:pass@broker.com:5671",
		},
		{
			name: "builds without TLS",
			config: Config{
				Host:     "broker.com",
				Username: "user",
				Password: "pass",
				UseTLS:   false,
			},
			want: "amqp://user:pass@broker.com:5672",
		},
		{
			name: "uses custom port",
			config: Config{
				Host:     "broker.com",
				Port:     9999,
				Username: "user",
				Password: "pass",
				UseTLS:   true,
			},
			want: "amqps://user:pass@broker.com:9999",
		},
		{
			name: "URL encodes password",
			config: Config{
				Host:     "broker.com",
				Username: "user",
				Password: "p@ss=word&special",
				UseTLS:   true,
			},
			want: "amqps://user:p%40ss%3Dword%26special@broker.com:5671",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.BuildConnectionString()
			if got != tt.want {
				t.Errorf("BuildConnectionString() = %v, want %v", got, tt.want)
			}
		})
	}
}

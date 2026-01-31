# gokyu

A cloud-agnostic message queue library for Go using AMQP 1.0 protocol. Switch between cloud providers without changing your application code.

## Supported Providers

| Provider | Service | Status |
|----------|---------|--------|
| Azure | Azure Service Bus | ✅ Supported |
| AWS | Amazon MQ (ActiveMQ) | ✅ Supported |

## Installation

```bash
go get github.com/venderneutral/gokyu
```

## Quick Start

```go
package main

import (
    "context"
    "log"

    "github.com/venderneutral/gokyu"
    _ "github.com/venderneutral/gokyu/providers" // Imports all providers
)

func main() {
    ctx := context.Background()

    // Create a client - switch providers by changing just this config
    client, err := gokyu.NewClient(&gokyu.Config{
        Provider:         gokyu.ProviderAzure, // or gokyu.ProviderAmazonMQ
        ConnectionString: "amqps://...",
        Topic:            "my-topic",
        Subscription:     "my-subscription",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Publish a message
    publisher, _ := client.NewPublisher(ctx)
    defer publisher.Close(ctx)
    
    msg := gokyu.NewMessage([]byte("Hello, World!"))
    publisher.Publish(ctx, msg)

    // Receive messages
    subscriber, _ := client.NewSubscriber(ctx)
    defer subscriber.Close(ctx)

    received, _ := subscriber.Receive(ctx)
    log.Printf("Received: %s", string(received.Body))
    subscriber.Ack(ctx, received)
}
```

## Configuration

### Programmatic Configuration

```go
client, err := gokyu.NewClient(&gokyu.Config{
    Provider:         gokyu.ProviderAzure,
    ConnectionString: "amqps://policy:key@namespace.servicebus.windows.net",
    Topic:            "my-topic",
    Subscription:     "my-subscription",
})
```

Or with individual parameters:

```go
client, err := gokyu.NewClient(&gokyu.Config{
    Provider:     gokyu.ProviderAmazonMQ,
    Host:         "broker-id.mq.us-east-1.amazonaws.com",
    Port:         5671,
    Username:     "admin",
    Password:     "password",
    UseTLS:       true,
    Queue:        "my-queue", // Use Queue for point-to-point
})
```

### Environment Variables

```bash
export GOKYU_PROVIDER=azure
export GOKYU_CONNECTION_STRING=amqps://...
export GOKYU_TOPIC=my-topic
export GOKYU_SUBSCRIPTION=my-subscription
```

```go
import (
    "github.com/venderneutral/gokyu"
    _ "github.com/venderneutral/gokyu/providers" // Imports all providers
)

client, err := gokyu.NewClientFromEnv()
```

| Variable | Description |
|----------|-------------|
| `GOKYU_PROVIDER` | Provider name: `azure` or `amazonmq` |
| `GOKYU_CONNECTION_STRING` | Full AMQP connection string |
| `GOKYU_HOST` | Broker hostname (if not using connection string) |
| `GOKYU_PORT` | Broker port (default: 5671) |
| `GOKYU_USERNAME` | Authentication username |
| `GOKYU_PASSWORD` | Authentication password |
| `GOKYU_QUEUE` | Queue name (for point-to-point) |
| `GOKYU_TOPIC` | Topic name (for pub/sub) |
| `GOKYU_SUBSCRIPTION` | Subscription name (for receiving from topics) |

## Provider-Specific Notes

### Azure Service Bus

Connection string format:
```
amqps://<policy-name>:<access-key>@<namespace>.servicebus.windows.net
```

Topic subscription addressing:
- Topics: `my-topic`
- Subscriptions: `my-topic/Subscriptions/my-subscription`

### Amazon MQ (ActiveMQ)

Connection string format:
```
amqps://<username>:<password>@<broker-id>.mq.<region>.amazonaws.com:5671
```

Uses ActiveMQ Virtual Topics for durable subscriptions:
- Queue: `my-queue`
- Topic subscription: `Consumer.<subscription>.VirtualTopic.<topic>`

## API Reference

### Message

```go
type Message struct {
    ID         string                 // Message identifier
    Body       []byte                 // Message payload
    Properties map[string]interface{} // Custom properties/headers
}

msg := gokyu.NewMessage([]byte("payload"))
msg.Properties["custom-header"] = "value"
```

### Publisher

```go
type Publisher interface {
    Publish(ctx context.Context, msg *Message) error
    Close(ctx context.Context) error
}
```

### Subscriber

```go
type Subscriber interface {
    Receive(ctx context.Context) (*Message, error)
    Ack(ctx context.Context, msg *Message) error   // Acknowledge success
    Nack(ctx context.Context, msg *Message) error  // Release for redelivery
    Close(ctx context.Context) error
}
```

## Error Handling

```go
import "errors"

msg, err := subscriber.Receive(ctx)
if errors.Is(err, gokyu.ErrConnectionFailed) {
    // Handle connection issues
}
if errors.Is(err, gokyu.ErrReceiveFailed) {
    // Handle receive failures
}
```

Common errors:
- `ErrConnectionFailed` - Connection to broker failed
- `ErrPublishFailed` - Message publish failed
- `ErrReceiveFailed` - Message receive failed
- `ErrAckFailed` - Message acknowledgment failed
- `ErrUnsupportedProvider` - Provider not registered

## Examples

See the [examples](./examples) directory:
- [Azure Service Bus](./examples/azure/main.go)
- [Amazon MQ](./examples/amazonmq/main.go)

## Switching Providers

One of gokyu's main goals is to make switching between cloud providers effortless. Here's how to do it:

### Option 1: Environment Variables (Zero Code Changes)

Import all providers once, then switch entirely via environment variables:

```go
import (
    "github.com/venderneutral/gokyu"
    _ "github.com/venderneutral/gokyu/providers" // Imports ALL providers
)

func main() {
    // Configuration comes entirely from environment
    client, err := gokyu.NewClientFromEnv()
    // ... rest of your code stays the same
}
```

**Switch from Azure to AWS by changing only environment variables:**

```bash
# Azure Service Bus
export GOKYU_PROVIDER=azure
export GOKYU_CONNECTION_STRING=amqps://policy:key@namespace.servicebus.windows.net
export GOKYU_TOPIC=orders
export GOKYU_SUBSCRIPTION=processor

# Amazon MQ (just change these values, no code changes needed)
export GOKYU_PROVIDER=amazonmq
export GOKYU_CONNECTION_STRING=amqps://admin:password@broker-id.mq.us-east-1.amazonaws.com:5671
export GOKYU_TOPIC=orders
export GOKYU_SUBSCRIPTION=processor
```

### Option 2: Configuration-Driven (Minimal Code Changes)

If you use programmatic configuration, switching requires changing only the `Provider` and `ConnectionString`:

```go
// Before: Azure
client, _ := gokyu.NewClient(&gokyu.Config{
    Provider:         gokyu.ProviderAzure,
    ConnectionString: os.Getenv("CONNECTION_STRING"),
    Topic:            "orders",
    Subscription:     "processor",
})

// After: Amazon MQ (only Provider and ConnectionString change)
client, _ := gokyu.NewClient(&gokyu.Config{
    Provider:         gokyu.ProviderAmazonMQ,
    ConnectionString: os.Getenv("CONNECTION_STRING"),
    Topic:            "orders",
    Subscription:     "processor",
})
```

### What Stays the Same

Your business logic never changes:

```go
// This code works identically on Azure and AWS
publisher, _ := client.NewPublisher(ctx)
publisher.Publish(ctx, gokyu.NewMessage([]byte("order created")))

subscriber, _ := client.NewSubscriber(ctx)
msg, _ := subscriber.Receive(ctx)
processOrder(msg.Body)
subscriber.Ack(ctx, msg)
```

### Migration Checklist

| Step | Environment Variables | Programmatic Config |
|------|----------------------|---------------------|
| 1. Import providers | `_ "github.com/venderneutral/gokyu/providers"` (all) | Change specific import |
| 2. Update provider | Change `GOKYU_PROVIDER` | Change `Provider` field |
| 3. Update connection | Change `GOKYU_CONNECTION_STRING` | Change `ConnectionString` |
| 4. Update topic/queue names | Change `GOKYU_TOPIC` etc. | Change `Topic` etc. |
| 5. Business logic | ✅ No changes | ✅ No changes |

## License

MIT License

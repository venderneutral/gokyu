// Example: Amazon MQ (ActiveMQ) Pub/Sub with goqueue
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/venderneutral/gokyu"
	_ "github.com/venderneutral/gokyu/providers/amazonmq" // Register Amazon MQ provider
)

func main() {
	logger := log.New(os.Stdout, "[goqueue-amazonmq] ", log.LstdFlags|log.Lshortfile)
	logger.Println("Starting Amazon MQ example")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create client with programmatic configuration
	// Amazon MQ connection string format: amqps://username:password@broker-id.mq.region.amazonaws.com:5671
	client, err := gokyu.NewClient(&gokyu.Config{
		Provider:         gokyu.ProviderAmazonMQ,
		ConnectionString: os.Getenv("AMAZONMQ_CONNECTION_STRING"),
		// Or use individual parameters:
		// Host:     os.Getenv("AMAZONMQ_HOST"),
		// Username: os.Getenv("AMAZONMQ_USERNAME"),
		// Password: os.Getenv("AMAZONMQ_PASSWORD"),
		// UseTLS:   true,
		Topic:        os.Getenv("AMAZONMQ_TOPIC"),
		Subscription: os.Getenv("AMAZONMQ_SUBSCRIPTION"),
	})
	if err != nil {
		logger.Fatalf("Failed to create client: %v", err)
	}

	// Create publisher
	publisher, err := client.NewPublisher(ctx)
	if err != nil {
		logger.Fatalf("Failed to create publisher: %v", err)
	}
	defer publisher.Close(ctx)

	// Create subscriber
	subscriber, err := client.NewSubscriber(ctx)
	if err != nil {
		logger.Fatalf("Failed to create subscriber: %v", err)
	}
	defer subscriber.Close(ctx)

	// Start subscriber in background
	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Println("Subscriber shutting down...")
				return
			default:
				msg, err := subscriber.Receive(ctx)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					logger.Printf("Receive error: %v", err)
					continue
				}

				logger.Printf("Received message: %s", string(msg.Body))

				if err := subscriber.Ack(ctx, msg); err != nil {
					logger.Printf("Ack error: %v", err)
				}
			}
		}
	}()

	// Publish a test message
	time.Sleep(time.Second) // Wait for subscriber to be ready

	msg := gokyu.NewMessage([]byte("Hello from goqueue on Amazon MQ!"))
	if err := publisher.Publish(ctx, msg); err != nil {
		logger.Printf("Publish error: %v", err)
	} else {
		logger.Println("Published message: Hello from goqueue on Amazon MQ!")
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Println("Shutdown signal received")
	cancel()
	time.Sleep(time.Second)
	logger.Println("Gracefully shut down")
}

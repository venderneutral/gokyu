// Example: Azure Service Bus Pub/Sub with goqueue
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/venderneutral/gokyu"
	_ "github.com/venderneutral/gokyu/providers/azure" // Register Azure provider
)

func main() {
	logger := log.New(os.Stdout, "[goqueue-azure] ", log.LstdFlags|log.Lshortfile)
	logger.Println("Starting Azure Service Bus example")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create client with programmatic configuration
	client, err := gokyu.NewClient(&gokyu.Config{
		Provider:         gokyu.ProviderAzure,
		ConnectionString: os.Getenv("ASB_CONNECTION_STRING"),
		Topic:            os.Getenv("ASB_TOPIC"),
		Subscription:     os.Getenv("ASB_SUBSCRIPTION"),
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

	msg := gokyu.NewMessage([]byte("Hello from goqueue!"))
	if err := publisher.Publish(ctx, msg); err != nil {
		logger.Printf("Publish error: %v", err)
	} else {
		logger.Println("Published message: Hello from goqueue!")
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

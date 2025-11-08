package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/webhook-processor/internal/adapters/rabbitmq"
	"github.com/webhook-processor/internal/domain"
	"github.com/webhook-processor/internal/ports"
)

func main() {
	log.Println("Starting Webhook Processor Producer...")

	logger := rabbitmq.NewConsoleLogger("PRODUCER")

	// Configuration for RabbitMQ connection
	config := ports.ConnectionConfig{
		Host:        getEnvOrDefault("RABBITMQ_HOST", "localhost"),
		Port:        5672,
		Username:    getEnvOrDefault("RABBITMQ_USER", "admin"),
		Password:    getEnvOrDefault("RABBITMQ_PASS", "password"),
		VirtualHost: getEnvOrDefault("RABBITMQ_VHOST", "/"),
		MaxRetries:  5,
		RetryDelay:  5,
		Heartbeat:   60,
		ConnTimeout: 30,
	}

	broker := rabbitmq.NewBroker(config, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := broker.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	// graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	publishCtx, publishCancel := context.WithCancel(context.Background())
	defer publishCancel()

	go func() {
		if err := publishMessages(publishCtx, broker, logger); err != nil {
			logger.Error("Error publishing messages", err)
		}
	}()

	<-sigChan
	logger.Info("Shutdown signal received, stopping producer...")

	publishCancel()

	time.Sleep(2 * time.Second)

	if err := broker.Close(); err != nil {
		logger.Error("Error closing broker connection", err)
	}

	logger.Info("Producer stopped successfully")
}

func publishMessages(ctx context.Context, broker ports.MessagePublisher, logger ports.Logger) error {
	// Define message templates
	messageTemplates := []struct {
		messageType string
		exchange    string
		routingKey  string
		payload     map[string]interface{}
	}{
		{
			messageType: "webhook",
			exchange:    "webhook.exchange",
			routingKey:  "webhook.process",
			payload: map[string]interface{}{
				"webhook_url": "https://api.example.com/webhook",
				"data": map[string]interface{}{
					"event":      "user.created",
					"user_id":    12345,
					"timestamp":  time.Now().UTC().Format(time.RFC3339),
					"ip_address": "192.168.1.100",
				},
				"headers": map[string]string{
					"Content-Type":  "application/json",
					"Authorization": "Bearer token123",
				},
			},
		},
		{
			messageType: "email",
			exchange:    "email.exchange",
			routingKey:  "email.send",
			payload: map[string]interface{}{
				"to":       "user@example.com",
				"from":     "noreply@webhook-processor.com",
				"subject":  "Welcome to Webhook Processor!",
				"body":     "Thank you for signing up. Your account has been created successfully.",
				"priority": "normal",
				"template": "welcome_email",
			},
		},
		{
			messageType: "email",
			exchange:    "email.exchange",
			routingKey:  "email.priority",
			payload: map[string]interface{}{
				"to":       "admin@example.com",
				"from":     "alerts@webhook-processor.com",
				"subject":  "High Priority: System Alert",
				"body":     "A critical system event has occurred and requires immediate attention.",
				"priority": "high",
				"alert":    true,
			},
		},
	}

	messageCount := 0
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Publishing context cancelled, stopping message publishing")
			return ctx.Err()

		case <-ticker.C:
			// Select a random message template
			template := messageTemplates[messageCount%len(messageTemplates)]
			messageCount++

			// Create a new message
			message := domain.NewMessage(template.messageType, template.payload)

			// Add some custom headers
			message.AddHeader("producer", "webhook-processor-producer")
			message.AddHeader("environment", getEnvOrDefault("ENVIRONMENT", "development"))
			message.AddHeader("version", "1.0.0")
			message.AddHeader("message_count", fmt.Sprintf("%d", messageCount))

			// Convert payload to JSON for logging
			payloadJSON, _ := json.Marshal(template.payload)
			logger.Info("Publishing message",
				"messageId", message.ID,
				"type", message.Type,
				"exchange", template.exchange,
				"routingKey", template.routingKey,
				"payloadSize", len(payloadJSON))

			// Publish the message
			if err := broker.Publish(ctx, template.exchange, template.routingKey, message); err != nil {
				logger.Error("Failed to publish message", err,
					"messageId", message.ID,
					"type", message.Type,
					"exchange", template.exchange,
					"routingKey", template.routingKey)
				continue
			}

			logger.Info("Message published successfully",
				"messageId", message.ID,
				"type", message.Type,
				"messageCount", messageCount)

			// Simulate different publishing patterns
			if messageCount%10 == 0 {
				// Every 10th message, publish a batch
				logger.Info("Publishing batch of messages", "batchSize", 3)
				if err := publishBatch(ctx, broker, logger, template, 3); err != nil {
					logger.Error("Failed to publish batch", err)
				}
			}

			if messageCount%25 == 0 {
				// Every 25th message, publish with custom options
				options := ports.PublishOptions{
					Mandatory: true,
					Headers: map[string]interface{}{
						"special":    true,
						"milestone":  messageCount,
						"batch_type": "milestone",
					},
				}

				milestoneMessage := domain.NewMessage("webhook", map[string]interface{}{
					"webhook_url": "https://api.example.com/milestone",
					"data": map[string]interface{}{
						"event":     "milestone.reached",
						"count":     messageCount,
						"timestamp": time.Now().UTC().Format(time.RFC3339),
					},
				})

				if err := broker.PublishWithOptions(ctx, template.exchange, template.routingKey, milestoneMessage, options); err != nil {
					logger.Error("Failed to publish milestone message", err, "messageId", milestoneMessage.ID)
				} else {
					logger.Info("Milestone message published", "messageId", milestoneMessage.ID, "count", messageCount)
				}
			}
		}
	}
}

func publishBatch(ctx context.Context, broker ports.MessagePublisher, logger ports.Logger, template struct {
	messageType string
	exchange    string
	routingKey  string
	payload     map[string]interface{}
}, batchSize int) error {

	for i := 0; i < batchSize; i++ {
		// Create batch message with modified payload
		batchPayload := make(map[string]interface{})
		for k, v := range template.payload {
			batchPayload[k] = v
		}
		batchPayload["batch_index"] = i
		batchPayload["batch_size"] = batchSize
		batchPayload["batch_id"] = fmt.Sprintf("batch_%d_%d", time.Now().Unix(), i)

		message := domain.NewMessage(template.messageType, batchPayload)
		message.AddHeader("batch", "true")
		message.AddHeader("batch_index", fmt.Sprintf("%d", i))

		if err := broker.Publish(ctx, template.exchange, template.routingKey, message); err != nil {
			return fmt.Errorf("failed to publish batch message %d: %w", i, err)
		}

		logger.Info("Batch message published", "messageId", message.ID, "batchIndex", i)

		// Small delay between batch messages
		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

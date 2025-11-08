package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	email_model "github.com/webhook-processor/internal/shared/email/model"
	log "github.com/webhook-processor/internal/shared/logger"
	"github.com/webhook-processor/shared/env"
)

func main() {
	logger := log.NewLogger(
		&log.NewLoggerOptions{
			Prefix: "CONSUMER",
			Level:  env.GetEnvOrDefault("LOG_LEVEL", "debug"),
		},
	)

	logger.Info("Starting Webhook Processor Consumer...")

	url := fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		"admin",
		"password",
		"localhost",
		5672,
		"/",
	)

	conn, err := amqp.DialConfig(url, amqp.Config{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, 2*time.Second)
		},
	})
	failOnError(err, "Failed to connect to RabbitMQ")

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		email_model.EMAIL_QUEUE, // name
		true,                    // durable
		false,                   // delete when unused
		false,                   // exclusive
		false,                   // no-wait
		nil,                     // arguments
	)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		q.Name,          // queue
		"test_consumer", // consumer
		false,           // auto-ack
		false,           // exclusive
		false,           // no supported
		false,           // no-wait
		nil,             // args
	)
	failOnError(err, "Failed to register a consumer")

	go func() {
		for d := range msgs {
			logger.Info("Received a message: %s", d.Body)
			err := d.Ack(false)
			if err != nil {
				logger.Error("Error acknowledging message", err)
			}
		}
	}()

	// go startHealthCheck(consumeCtx, broker, logger)

	logger.Info("waiting for messages...")

	// graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	logger.Info("Shutdown signal received, stopping consumer...")

	shutdownTimeout := 10 * time.Second
	logger.Info("Waiting for graceful shutdown", "timeout", shutdownTimeout)
	time.Sleep(shutdownTimeout)

	if err := conn.Close(); err != nil {
		logger.Error("Error closing broker connection", err)
	}

	logger.Info("Consumer stopped successfully")
}

// func startHealthCheck(ctx context.Context, broker ports.MessageBroker, logger ports.Logger) {
// 	ticker := time.NewTicker(30 * time.Second)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			logger.Info("Health check routine stopped")
// 			return

// 		case <-ticker.C:
// 			if !broker.IsConnected() {
// 				logger.Error("Health check failed: broker not connected", nil)
// 			} else {
// 				logger.Debug("Health check passed: broker connected")
// 			}
// 		}
// 	}
// }

func failOnError(err error, msg string) {
	if err != nil {
		panic(err)
	}
}

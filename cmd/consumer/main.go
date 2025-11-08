package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	wb "github.com/webhook-processor/internal/webhook/domain"
	wb_usecase "github.com/webhook-processor/internal/webhook/usecase"

	amqp "github.com/rabbitmq/amqp091-go"
	env "github.com/webhook-processor/internal/shared/env"
	log "github.com/webhook-processor/internal/shared/logger"
)

func main() {
	logger := log.NewLogger(
		&log.NewLoggerOptions{
			Prefix: "CONSUMER",
			Level:  env.GetEnvOrDefault("LOG_LEVEL", "debug"),
		},
	)
	logger.SetAsDefaultForPackage()

	log.Info("Starting Webhook Processor Consumer...")

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
		wb.WEBHOOK_QUEUE, // name
		true,             // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
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
			log.Info("Received a message: %s", d.Body)
			wbEvent := wb.WebhookEventMessage{}
			err := json.Unmarshal(d.Body, &wbEvent)
			if err != nil {
				err = d.Ack(false)
				if err != nil {
					log.Error("Error acknowledging message", err)
				}
				continue
			}
			success, _ := wb_usecase.SendWebhook(wbEvent)
			if success {
				fmt.Println("Webhook sent successfully")
				err = d.Ack(false)
				if err != nil {
					log.Error("Error acknowledging message", err)
				}
			} else {
				err = d.Nack(false, true)
				if err != nil {
					log.Error("Error nacking message", err)
				}
			}
		}
	}()

	go startHealthCheck(ch)

	log.Info("waiting for messages...")

	// graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Info("Shutdown signal received, stopping consumer...")

	shutdownTimeout := 10 * time.Second
	log.Info("Waiting for graceful shutdown", "timeout", shutdownTimeout)
	time.Sleep(shutdownTimeout)

	if err := conn.Close(); err != nil {
		log.Error("Error closing broker connection", err)
	}

	log.Info("Consumer stopped successfully")
}

func startHealthCheck(ch *amqp.Channel) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		if ch.IsClosed() {
			log.Error("Health check failed: broker not connected", nil)
		} else {
			log.Debug("Health check passed: broker connected")
		}
	}
}

func failOnError(err error, msg string) {
	if err != nil {
		panic(err)
	}
}

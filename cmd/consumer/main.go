package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/webhook-processor/internal/webhook/adapters/queue"
	wb_queue "github.com/webhook-processor/internal/webhook/adapters/queue"
	wb_repo "github.com/webhook-processor/internal/webhook/adapters/repo"
	wb_model "github.com/webhook-processor/internal/webhook/domain/model"
	wb "github.com/webhook-processor/internal/webhook/domain/service"

	env "github.com/webhook-processor/internal/shared/env"
	"github.com/webhook-processor/internal/shared/http"
	log "github.com/webhook-processor/internal/shared/logger"
	"github.com/webhook-processor/internal/shared/persistence/gorm"
)

func main() {
	logger := log.NewLogger(
		&log.NewLoggerOptions{
			Prefix: "CONSUMER",
			Level:  env.GetEnvOrDefault("LOG_LEVEL", "debug"),
		},
	)
	logger.SetAsDefaultForPackage()

	db := gorm.NewDB(gorm.DbOptions{
		Host:     env.GetEnvOrDefault("POSTGRES_HOST", "localhost"),
		DbName:   env.GetEnvOrDefault("POSTGRES_DB", "webhook_processor"),
		User:     env.GetEnvOrDefault("POSTGRES_USER", "webhook_user"),
		Password: env.GetEnvOrDefault("POSTGRES_PASSWORD", "webhook_pass"),
		Schema:   env.GetEnvOrDefault("POSTGRES_SCHEMA", "webhooks"),
	})

	log.Info("Starting Webhook Processor Consumer...")

	connector := queue.NewRabbitMQConnector(&queue.RabbitMQConnOpts{
		Queue_name:    wb_model.WEBHOOK_QUEUE,
		Exchange_name: wb_model.EXCHANGE_NAME,
	})

	repo := wb_repo.NewWebhookRepo(db)
	http_client := http.NewClient(http.ClientOpts{Timeout: time.Second * 5})
	wb_service := wb.NewWebhookService(repo, http_client)
	rabbitMQConsumer := wb_queue.NewRabbitMQConsumer(wb_service)

	msgs := connector.Listen()
	go func() {
		for d := range msgs {
			rabbitMQConsumer.Consume(d)
		}
	}()

	log.Info("waiting for messages...")

	// graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Info("Shutdown signal received, stopping consumer...")

	shutdownTimeout := 3 * time.Second
	log.Info("Waiting for graceful shutdown", "timeout", shutdownTimeout)
	time.Sleep(shutdownTimeout)

	if err := connector.Close(); err != nil {
		log.Error("Error closing broker connection", err)
	}

	log.Info("Consumer stopped successfully")
}

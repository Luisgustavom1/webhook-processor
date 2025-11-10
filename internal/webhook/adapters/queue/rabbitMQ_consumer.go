package queue

import (
	"encoding/json"

	"github.com/rabbitmq/amqp091-go"
	"github.com/webhook-processor/internal/shared/logger"
	"github.com/webhook-processor/internal/webhook/ports"

	wb_model "github.com/webhook-processor/internal/webhook/domain/model"
)

type RabbitMQConsumer struct {
	service ports.WebhookServicePort
}

func NewRabbitMQConsumer(service ports.WebhookServicePort) *RabbitMQConsumer {
	return &RabbitMQConsumer{service: service}
}

func (c *RabbitMQConsumer) Consume(msg amqp091.Delivery) error {
	logger.Info("Received a message", "msg", msg.Body)
	wbEvent := wb_model.WebhookEventMessage{}
	err := json.Unmarshal(msg.Body, &wbEvent)
	if err != nil {
		return ack(msg)
	}

	wb_error := c.service.SendWebhook(wbEvent)
	if wb_error != nil {
		logger.Error("Error sending webhook", err)
		if wb_error.IsRetryable() {
			return ack(msg)
		}

		return nack(msg)
	}

	return nil
}

func ack(msg amqp091.Delivery) error {
	err := msg.Ack(false)
	if err != nil {
		logger.Error("Error acknowledging message", err)
	}
	return err
}

func nack(msg amqp091.Delivery) error {
	err := msg.Nack(false, true)
	if err != nil {
		logger.Error("Error acknowledging message", err)
	}
	return err
}

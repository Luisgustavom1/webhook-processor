package queue

import (
	"encoding/json"

	"github.com/rabbitmq/amqp091-go"
	log "github.com/webhook-processor/internal/shared/logger"
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
	log.Info("Received a message", "msg", msg.Body)
	wbEvent := wb_model.WebhookEventMessage{}
	err := json.Unmarshal(msg.Body, &wbEvent)
	if err != nil {
		return ack(msg)
	}

	_, wb_error := c.service.SendWebhook(wbEvent)

	if wb_error != nil {
		log.Debug(wb_error.Error())
		if wb_error.IsRetryable() {
			return nack(msg)
		}

		return ack(msg)
	}

	return ack(msg)
}

func ack(msg amqp091.Delivery) error {
	log.Debug("acknowledging message")
	err := msg.Ack(false)
	if err != nil {
		log.Error("Error acknowledging message", err)
	}
	return err
}

func nack(msg amqp091.Delivery) error {
	log.Debug("nacknowledging message")
	err := msg.Nack(false, true)
	if err != nil {
		log.Error("Error nacknowledging message", err)
	}
	return err
}

package queue

import (
	"context"
	"encoding/json"
	"math"
	"math/rand"

	"github.com/rabbitmq/amqp091-go"
	log "github.com/webhook-processor/internal/shared/logger"
	"github.com/webhook-processor/internal/webhook/ports"

	wb_model "github.com/webhook-processor/internal/webhook/domain/model"
)

type RabbitMQConsumer struct {
	service ports.WebhookServicePort
	queue   ports.QueuePort
}

const MAX_DELAY = 60000

func NewRabbitMQConsumer(service ports.WebhookServicePort, queue ports.QueuePort) *RabbitMQConsumer {
	return &RabbitMQConsumer{service: service, queue: queue}
}

func (c *RabbitMQConsumer) Consume(msg amqp091.Delivery) error {
	log.Info("Received a message", "msg", msg.Body)
	wbEvent := wb_model.WebhookEventMessage{}
	err := json.Unmarshal(msg.Body, &wbEvent)
	if err != nil {
		return ack(msg)
	}

	ctx := context.Background()
	wb_event, wb_error := c.service.SendWebhook(ctx, wbEvent)

	if wb_error != nil && wb_error.IsRetryable() {
		log.Info(wb_error.Error())
		delay := getDelay(wb_event.Tries)
		log.Info("publishing message with delay", "delay", delay)
		err := c.queue.Publish(ctx, msg.Body, ports.QueuePortPublishOpts{Delay: delay})
		if err != nil {
			log.Error("Error publishing message", err)
			return err
		}
	}

	return ack(msg)
}

func ack(msg amqp091.Delivery) error {
	log.Debug("acknowledging message")
	msg.Headers["x-delay"] = 4000
	err := msg.Ack(false)
	if err != nil {
		log.Error("Error acknowledging message", err)
	}
	return err
}

func getDelay(retryCount int) int {
	// 2 ^ 0 = 1
	// 2 ^ 1 = 2
	// 2 ^ 2 = 4
	// 2 ^ 3 = 8
	// 2 ^ 4 = 16
	exp := math.Pow(2, float64(retryCount))
	delay := math.Min(exp*1000, MAX_DELAY)
	jitter := rand.Float64() * (delay * 0.5)

	return int(delay) + int(jitter)
}

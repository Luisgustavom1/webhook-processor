package queue

import (
	"fmt"
	"net"
	"time"

	wb_model "github.com/webhook-processor/internal/webhook/domain/model"

	amqp "github.com/rabbitmq/amqp091-go"
	log "github.com/webhook-processor/internal/shared/logger"
)

type RabbitMQConnector struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	q    amqp.Queue
}

type RabbitMQConnOpts struct {
	Queue_name    string
	Exchange_name string
}

func NewRabbitMQConnector(opts *RabbitMQConnOpts) *RabbitMQConnector {
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

	err = ch.ExchangeDeclare(opts.Exchange_name, "x-delayed-message", true, false, false, false, amqp.Table{
		"x-delayed-type": "fanout",
	})
	failOnError(err, "failed to declare exchange")

	err = ch.QueueBind(opts.Queue_name, "webhook.process", opts.Exchange_name, false, nil)

	q, err := ch.QueueDeclare(
		opts.Queue_name, // name
		true,            // durable
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		nil,
	)
	failOnError(err, "Failed to declare a queue")

	return &RabbitMQConnector{
		conn: conn,
		ch:   ch,
		q:    q,
	}
}

func (l *RabbitMQConnector) Listen() <-chan amqp.Delivery {
	go startHealthCheck(l.ch)

	msgs, err := l.ch.Consume(
		l.q.Name, // queue
		fmt.Sprintf("consumer::%s", wb_model.WEBHOOK_QUEUE), // consumer
		false, // auto-ack
		false, // exclusive
		false, // no supported
		false, // no-wait
		nil,   // args
	)
	failOnError(err, "Failed to register a consumer")

	return msgs
}

func (l *RabbitMQConnector) Close() error {
	err := l.ch.Close()
	err = l.conn.Close()
	return err
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

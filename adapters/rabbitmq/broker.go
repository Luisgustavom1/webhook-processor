package rabbitmq

import "fmt"

func NewBroker(config ConnectionConfig, logger Logger) *Broker {
	return &Broker{
		config: config,
		logger: logger,
	}
}

func (b *Broker) Connect() error {
	conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%d", b.config.Host, b.config.Port))
	if err != nil {
		return err
	}
	b.conn = conn
	return nil
}

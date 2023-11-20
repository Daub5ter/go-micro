package event

import (
	"errors"
	amqp "github.com/rabbitmq/amqp091-go"
)

func declareExchange(name string, ch *amqp.Channel) error {
	switch name {
	case "logs_topic", "auth", "get_by_email":
		return ch.ExchangeDeclare(
			name,
			"topic",
			true,
			false,
			false,
			false,
			nil,
		)
	default:
		return errors.New("invalid name of channel RabbitMQ")
	}
}

func declareRandomQueue(ch *amqp.Channel) (amqp.Queue, error) {
	return ch.QueueDeclare(
		"",
		false,
		false,
		true,
		false,
		nil,
	)
}

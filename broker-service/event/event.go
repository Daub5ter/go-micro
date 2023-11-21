package event

import (
	"errors"
	amqp "github.com/rabbitmq/amqp091-go"
)

func declareExchange(name string, ch *amqp.Channel) error {
	switch name {
	case "log", "mail", "authenticate_user", "get_user_by_email", "get_user_by_id":
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

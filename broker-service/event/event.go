package event

import (
	"errors"
	amqp "github.com/rabbitmq/amqp091-go"
)

func declareExchange(name string, ch *amqp.Channel) error {
	switch name {
	case "log", "mail", "authenticate_user", "get_user_by_email", "get_user_by_id", "get_all_users", "registration_user",
		"update_user", "change_password", "delete_user_by_email", "delete_user_by_id", "authenticate_user_session":
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

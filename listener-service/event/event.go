package event

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

func declareExchange(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare("log", "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare("mail", "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare("authenticate_user", "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare("get_user_by_email", "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.ExchangeDeclare("get_user_by_id", "topic", true, false, false, false, nil); err != nil {
		return err
	}

	return nil
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

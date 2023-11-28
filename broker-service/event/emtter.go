package event

import (
	"broker/tools"
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
)

type Emitter struct {
	connection *amqp.Connection
}

func (e *Emitter) setup(name string) error {
	channel, err := e.connection.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	return declareExchange(name, channel)
}

func (e *Emitter) Push(event string, exchange string, severity string) error {
	channel, err := e.connection.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	log.Println("Pushing to channel")

	err = channel.PublishWithContext(
		context.TODO(),
		exchange,
		severity,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(event),
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func (e *Emitter) PushWithResponse(event string, exchange string, severity string) ([]byte, error) {
	var payload []byte

	channel, err := e.connection.Channel()
	if err != nil {
		return nil, err
	}
	defer channel.Close()

	q, err := declareRandomQueue(channel)

	msgs, err := channel.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	corrID := tools.RandomString(32)

	err = channel.PublishWithContext(
		context.TODO(),
		exchange,
		severity,
		false,
		false,
		amqp.Publishing{
			ContentType:   "text/plain",
			CorrelationId: corrID,
			ReplyTo:       q.Name,
			Body:          []byte(event),
		},
	)
	if err != nil {
		return nil, err
	}

	for d := range msgs {
		if corrID == d.CorrelationId {
			payload = d.Body
			if err != nil {
				return nil, err
			}
			break
		}
	}

	return payload, nil
}

func NewEventEmitter(name string, conn *amqp.Connection) (Emitter, error) {
	emitter := Emitter{
		connection: conn,
	}

	err := emitter.setup(name)
	if err != nil {
		return Emitter{}, err
	}

	return emitter, nil
}

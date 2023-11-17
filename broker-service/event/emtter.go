package event

import (
	"broker/tools"
	"context"
	"encoding/json"
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

func (e *Emitter) PushWithResponse(event string, exchange string, severity string) (string, error) {
	payload := struct {
		Error error
		Data  string
	}{}

	channel, err := e.connection.Channel()
	if err != nil {
		return "", err
	}
	defer channel.Close()

	q, err := declareRandomQueue(channel)

	msgs, err := channel.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return "", err
	}

	corrID := tools.RandomString(32)

	log.Println("Pushing to channel")

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
		return "", err
	}

	for d := range msgs {
		if corrID == d.CorrelationId {
			_ = json.Unmarshal(d.Body, &payload)
			if err != nil {
				return "", err
			}
			break
		}
	}

	return payload.Data, payload.Error
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

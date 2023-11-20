package event

import (
	"broker/tools"
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
)

type jsonResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type RequestPayload struct {
	Action     string            `json:"action"`
	Auth       AuthPayload       `json:"auth,omitempty"`
	Reg        RegPayload        `json:"reg,omitempty"`
	Update     UpdatePayload     `json:"update,omitempty"`
	ChPassword ChPasswordPayload `json:"change_password,omitempty"`
	Email      EmailPayload      `json:"get_by_email,omitempty"`
	ID         IDPayload         `json:"get_by_id,omitempty"`
	Log        LogPayload        `json:"log,omitempty"`
	Mail       MailPayload       `json:"mail,omitempty"`
}

type MailPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegPayload struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Password  string `json:"password"`
	Active    int    `json:"active"`
}

type UpdatePayload struct {
	Email       string `json:"email"`
	EmailChange string `json:"email_change"`
	FirstName   string `json:"first_name,omitempty,omitempty"`
	LastName    string `json:"last_name,omitempty,omitempty"`
	Active      int    `json:"active,omitempty"`
}

type ChPasswordPayload struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	NewPassword string `json:"new_password"`
}

type EmailPayload struct {
	Email string `json:"email"`
}

type IDPayload struct {
	ID int `json:"id"`
}

type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type Emitter struct {
	connection *amqp.Connection
}

type Payload struct {
	Error error  `json:"error"`
	Data  string `json:"data"`
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

package event

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"log"
	"net/http"
)

type Consumer struct {
	conn      *amqp.Connection
	queueName string
}

type jsonResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type Payload struct {
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

//type PayloadResponse struct {
//	Error error  `json:"error"`
//	Data  string `json:"data"`
//}

func NewConsumer(conn *amqp.Connection) (Consumer, error) {
	consumer := Consumer{
		conn: conn,
	}

	err := consumer.setup()
	if err != nil {
		return Consumer{}, err
	}

	return consumer, nil
}

func (consumer *Consumer) setup() error {
	channel, err := consumer.conn.Channel()
	if err != nil {
		return err
	}

	return declareExchange(channel)
}

func (consumer *Consumer) Listen() error {
	ch, err := consumer.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	q, err := declareRandomQueue(ch)
	if err != nil {
		return err
	}

	ch.QueueBind(
		q.Name,
		"log",
		"logs_topic",
		false,
		nil,
	)

	ch.QueueBind(
		q.Name,
		"auth",
		"auth",
		false,
		nil,
	)

	ch.QueueBind(
		q.Name,
		"get.by.email",
		"get_by_email",
		false,
		nil,
	)

	if err != nil {
		return err
	}

	messages, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		return err
	}

	forever := make(chan bool)
	go func() {
		for d := range messages {
			var payload Payload
			err = json.Unmarshal(d.Body, &payload)
			if err != nil {
				log.Println(err)
			}

			// go handlePayload(payload)
			response := handlePayload(payload)

			jsonResp, err := json.MarshalIndent(response, "", "\t")
			if err != nil {
				log.Println(err)
			}

			err = ch.PublishWithContext(
				context.TODO(),
				"",
				d.ReplyTo,
				false,
				false,
				amqp.Publishing{
					ContentType:   "text/plain",
					CorrelationId: d.CorrelationId,
					Body:          jsonResp,
				})
			if err != nil {
				log.Println(err)
			}
		}
	}()

	fmt.Printf("Waiting for messages")
	<-forever

	return nil
}

func handlePayload(payload Payload) jsonResponse {
	response := jsonResponse{}

	switch payload.Action {
	case "log":
		// log whatever we get
		err := logEvent(payload)
		if err != nil {
			log.Println(err)
		}

	case "auth":
		// authenticate
		resp, err := authEvent(payload)
		if err != nil {
			log.Println(err)
		}
		response = resp

	case "get_by_email":
		resp, err := getUserByEmailEvent(payload)
		if err != nil {
			log.Println(err)
		}
		response = resp

	default:
		errString := fmt.Sprintf("invalid name of function %s, RabbitMQ", payload.Action)
		log.Println(errString)
	}

	return response
}

func logEvent(entry Payload) error {
	// create some json we'll send to the auth microservice
	jsonData, _ := json.MarshalIndent(entry.Log, "", "\t")

	// call the service
	request, err := http.NewRequest("POST", "http://logger-service/log", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// make sure we get back the correct status code
	if response.StatusCode != http.StatusAccepted {
		return errors.New("service don`t work")
	}

	return nil
}

func authEvent(entry Payload) (jsonResponse, error) {
	// create some json we'll send to the auth microservice
	jsonData, err := json.MarshalIndent(entry.Auth, "", "\t")
	if err != nil {
		return jsonResponse{Error: true, Message: fmt.Sprintf("error %v", err)}, err
	}

	// call the service
	request, err := http.NewRequest("POST", "http://authentication-service/authenticate", bytes.NewBuffer(jsonData))
	if err != nil {
		return jsonResponse{Error: true, Message: fmt.Sprintf("error %v", err)}, err
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		return jsonResponse{Error: true, Message: fmt.Sprintf("%v", err)}, err
	}
	defer response.Body.Close()

	// make sure we get back the correct status code
	if response.StatusCode == http.StatusUnauthorized {
		err = errors.New("unauthorized")
		return jsonResponse{Error: true, Message: fmt.Sprintf("%v", err)}, err
	} else if response.StatusCode != http.StatusOK {
		err = errors.New("service don`t work")
		return jsonResponse{Error: true, Message: fmt.Sprintf("%v", err)}, err
	}

	jsonService := jsonResponse{}
	err = json.NewDecoder(response.Body).Decode(&jsonService)
	if err != nil {
		return jsonResponse{Error: true, Message: fmt.Sprintf("%v", err)}, err
	}

	return jsonService, nil
}

func getUserByEmailEvent(entry Payload) (jsonResponse, error) {
	// create some json we'll send to the auth microservice
	jsonData, err := json.MarshalIndent(entry.Email, "", "\t")
	if err != nil {
		return jsonResponse{Error: true, Message: fmt.Sprintf("error %v", err)}, err
	}

	// call the service
	request, err := http.NewRequest("POST", "http://authentication-service/get_by_email", bytes.NewBuffer(jsonData))
	if err != nil {
		return jsonResponse{Error: true, Message: fmt.Sprintf("error %v", err)}, err
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		return jsonResponse{Error: true, Message: fmt.Sprintf("%v", err)}, err
	}
	defer response.Body.Close()

	// make sure we get back the correct status code
	if response.StatusCode == http.StatusUnauthorized {
		err = errors.New("unauthorized")
		return jsonResponse{Error: true, Message: fmt.Sprintf("%v", err)}, err
	} else if response.StatusCode != http.StatusOK {
		err = errors.New("service don`t work")
		return jsonResponse{Error: true, Message: fmt.Sprintf("%v", err)}, err
	}

	jsonService := jsonResponse{}
	err = json.NewDecoder(response.Body).Decode(&jsonService)
	if err != nil {
		return jsonResponse{Error: true, Message: fmt.Sprintf("%v", err)}, err
	}

	return jsonService, nil
}

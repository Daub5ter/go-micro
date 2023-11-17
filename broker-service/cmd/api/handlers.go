package main

import (
	"broker/event"
	"broker/logs"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net/http"
	"net/rpc"
	"time"
)

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

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "Hit the broker",
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// HandleSubmission is the main point of entry into the broker. It accepts a JSON
// payload and performs an action based on the value of "action" in that JSON.
func (app *Config) HandleSubmission(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	switch requestPayload.Action {
	case "auth":
		app.authenticate(w, requestPayload.Auth)
	case "reg":
		app.registrate(w, requestPayload.Reg)
	case "update":
		app.update(w, requestPayload.Update)
	case "change_password":
		app.changePassword(w, requestPayload.ChPassword)
	case "get_all":
		app.getAll(w)
	case "get_by_email":
		app.getByEmail(w, requestPayload.Email)
	case "get_by_id":
		app.getByID(w, requestPayload.ID)
	case "get_by_email_delete":
		app.getByEmailDelete(w, requestPayload.Email)
	case "get_by_id_delete":
		app.getByIDDelete(w, requestPayload.ID)
	case "log":
		app.logItemViaRpc(w, requestPayload.Log)
	case "mail":
		app.sendMail(w, requestPayload.Mail)

	default:
		app.errorJSON(w, errors.New("unknown action"))
	}
}

// calls the authentication microservice and sends back the appropriate response
func (app *Config) authenticate(w http.ResponseWriter, a AuthPayload) {
	// create some json we'll send to the auth microservice
	jsonData, _ := json.MarshalIndent(a, "", "\t")

	// call the service
	request, err := http.NewRequest("POST", "http://authentication-service/authenticate", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	// make sure we get back the correct status code
	if response.StatusCode == http.StatusUnauthorized {
		app.errorJSON(w, errors.New("invalid credentials"))
		return
	} else if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New("error calling auth service"))
		return
	}

	// create a variable we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the auth service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if jsonFromService.Error {
		app.errorJSON(w, err, http.StatusUnauthorized)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Authenticated"
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) update(w http.ResponseWriter, u UpdatePayload) {
	// create some json we'll send to the auth microservice
	jsonData, _ := json.MarshalIndent(u, "", "\t")

	// call the service
	request, err := http.NewRequest("PUT", "http://authentication-service/update", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	// make sure we get back the correct status code
	if response.StatusCode == http.StatusUnauthorized {
		app.errorJSON(w, errors.New("invalid credentials"))
		return
	} else if response.StatusCode != http.StatusOK {
		app.errorJSON(w, errors.New("error calling auth service"))
		return
	}

	// create a variable we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the auth service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if jsonFromService.Error {
		app.errorJSON(w, err, http.StatusUnauthorized)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Updated"
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) changePassword(w http.ResponseWriter, cp ChPasswordPayload) {
	// create some json we'll send to the auth microservice
	jsonData, _ := json.MarshalIndent(cp, "", "\t")

	// call the service
	request, err := http.NewRequest("PUT", "http://authentication-service/change_password", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	// make sure we get back the correct status code
	if response.StatusCode == http.StatusUnauthorized {
		app.errorJSON(w, errors.New("invalid credentials"))
		return
	} else if response.StatusCode != http.StatusOK {
		app.errorJSON(w, errors.New("error calling auth service"))
		return
	}

	// create a variable we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the auth service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if jsonFromService.Error {
		app.errorJSON(w, err, http.StatusUnauthorized)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Password changed"
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) getAll(w http.ResponseWriter) {
	// call the service
	request, err := http.NewRequest("GET", "http://authentication-service/get_all", nil)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	// make sure we get back the correct status code
	if response.StatusCode == http.StatusUnauthorized {
		app.errorJSON(w, errors.New("invalid credentials"))
		return
	} else if response.StatusCode != http.StatusOK {
		app.errorJSON(w, errors.New("error calling auth service"))
		return
	}

	// create a variable we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the auth service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if jsonFromService.Error {
		app.errorJSON(w, err, http.StatusUnauthorized)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Received users"
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) getByEmail(w http.ResponseWriter, e EmailPayload) {
	// create some json we'll send to the auth microservice
	jsonData, _ := json.MarshalIndent(e, "", "\t")

	// call the service
	request, err := http.NewRequest("POST", "http://authentication-service/get_by_email", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	// make sure we get back the correct status code
	if response.StatusCode == http.StatusUnauthorized {
		app.errorJSON(w, errors.New("invalid credentials"))
		return
	} else if response.StatusCode != http.StatusOK {
		app.errorJSON(w, errors.New("error calling auth service"))
		return
	}

	// create a variable we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the auth service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if jsonFromService.Error {
		app.errorJSON(w, err, http.StatusUnauthorized)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Received user"
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) getByID(w http.ResponseWriter, i IDPayload) {
	// create some json we'll send to the auth microservice
	jsonData, _ := json.MarshalIndent(i, "", "\t")

	// call the service
	request, err := http.NewRequest("POST", "http://authentication-service/get_by_id", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	// make sure we get back the correct status code
	if response.StatusCode == http.StatusUnauthorized {
		app.errorJSON(w, errors.New("invalid credentials"))
		return
	} else if response.StatusCode != http.StatusOK {
		app.errorJSON(w, errors.New("error calling auth service"))
		return
	}

	// create a variable we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the auth service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if jsonFromService.Error {
		app.errorJSON(w, err, http.StatusUnauthorized)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Received user"
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) getByEmailDelete(w http.ResponseWriter, e EmailPayload) {
	// create some json we'll send to the auth microservice
	jsonData, _ := json.MarshalIndent(e, "", "\t")

	// call the service
	request, err := http.NewRequest("DELETE", "http://authentication-service/get_by_email_delete", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	// make sure we get back the correct status code
	if response.StatusCode == http.StatusUnauthorized {
		app.errorJSON(w, errors.New("invalid credentials"))
		return
	} else if response.StatusCode != http.StatusOK {
		app.errorJSON(w, errors.New("error calling auth service"))
		return
	}

	// create a variable we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the auth service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if jsonFromService.Error {
		app.errorJSON(w, err, http.StatusUnauthorized)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Deleted user"
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) getByIDDelete(w http.ResponseWriter, i IDPayload) {
	// create some json we'll send to the auth microservice
	jsonData, _ := json.MarshalIndent(i, "", "\t")

	// call the service
	request, err := http.NewRequest("DELETE", "http://authentication-service/get_by_id_delete", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	// make sure we get back the correct status code
	if response.StatusCode == http.StatusUnauthorized {
		app.errorJSON(w, errors.New("invalid credentials"))
		return
	} else if response.StatusCode != http.StatusOK {
		app.errorJSON(w, errors.New("error calling auth service"))
		return
	}

	// create a variable we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the auth service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if jsonFromService.Error {
		app.errorJSON(w, err, http.StatusUnauthorized)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Deleted user"
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) registrate(w http.ResponseWriter, r RegPayload) {
	// create some json we'll send to the auth microservice
	jsonData, _ := json.MarshalIndent(r, "", "\t")

	// call the service
	request, err := http.NewRequest("POST", "http://authentication-service/registrate", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	// make sure we get back the correct status code
	if response.StatusCode == http.StatusUnauthorized {
		app.errorJSON(w, errors.New("invalid credentials"))
		return
	} else if response.StatusCode != http.StatusCreated {
		app.errorJSON(w, errors.New("error calling auth service"))
		return
	}

	// create a variable we'll read response.Body into
	var jsonFromService jsonResponse

	// decode the json from the auth service
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if jsonFromService.Error {
		app.errorJSON(w, err, http.StatusUnauthorized)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Registrated"
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusCreated, payload)
}

// cals the logger microservice and sends back the appropriate response
func (app *Config) logItem(w http.ResponseWriter, entry LogPayload) {
	// create some json we'll send to the auth microservice
	jsonData, _ := json.MarshalIndent(entry, "", "\t")

	// call the service
	logServiceURL := "http://logger-service/log"

	request, err := http.NewRequest("POST", logServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	// make sure we get back the correct status code
	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, err)
		return
	}

	// create json response
	var payload jsonResponse
	payload.Error = false
	payload.Message = "Logged"

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) sendMail(w http.ResponseWriter, msg MailPayload) {
	jsonData, _ := json.MarshalIndent(msg, "", "\t")

	// call the mail service
	mailServiceURL := "http://mailer-service/send"

	// post to mail service
	request, err := http.NewRequest("POST", mailServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	// make sure we get back the right status code
	if response.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New("error calling mail service"))
		return
	}

	// send back json
	var payload jsonResponse
	payload.Error = false
	payload.Message = "Message sent to " + msg.To

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) logEventViaRabbit(w http.ResponseWriter, l LogPayload) {
	var name = "logs_topic"
	var requestPayload RequestPayload
	requestPayload.Action = "log"
	requestPayload.Log = l

	err := app.pushToQueue(name, requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "logged via RabbitMQ"

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) authEventViaRabbit(w http.ResponseWriter, a AuthPayload) {
	var name = "auth"
	var requestPayload RequestPayload
	requestPayload.Action = "auth"
	requestPayload.Auth = a

	err := app.pushToQueue(name, requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Authenticated via RabbitMQ"

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) pushToQueue(name string, payload RequestPayload) error {
	emitter, err := event.NewEventEmitter(name, app.Rabbit)
	if err != nil {
		return err
	}

	var j []byte

	switch name {
	case "logs_topic":
		j, _ = json.MarshalIndent(&payload, "", "\t")
		err = emitter.Push(string(j), name, "log")
	case "auth":
		j, _ = json.MarshalIndent(&payload, "", "\t")
		err = emitter.Push(string(j), name, "auth")
	default:
		log.Printf("invalid name of channel RabbitMQ %s", name)
	}

	if err != nil {
		return err
	}

	return nil
}

type RPCPayload struct {
	Name string
	Data string
}

func (app *Config) logItemViaRpc(w http.ResponseWriter, l LogPayload) {
	client, err := rpc.Dial("tcp", "logger-service:5001")
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	rpcPayload := RPCPayload{
		Name: l.Name,
		Data: l.Name,
	}

	var result string
	err = client.Call("RPCServer.LogInfo", rpcPayload, &result)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: result,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) LogViaGRPC(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	conn, err := grpc.Dial("logger-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer conn.Close()

	c := logs.NewLogServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = c.WriteLog(ctx, &logs.LogRequest{
		LogEntry: &logs.Log{
			Name: requestPayload.Log.Name,
			Data: requestPayload.Log.Data,
		},
	})
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "logged"

	app.writeJSON(w, http.StatusAccepted, payload)
}

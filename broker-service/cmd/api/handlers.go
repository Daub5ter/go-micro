package main

import (
	"broker/event"
	"broker/logs"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net/http"
	"net/rpc"
	"time"
)

// RequestPayload is basic structure to indicate action and data`s structure
type RequestPayload struct {
	Action         string                `json:"action"`
	Auth           AuthUserPayload       `json:"auth,omitempty"`
	Reg            RegUserPayload        `json:"reg,omitempty"`
	UpdateUser     UpdateUserPayload     `json:"update_user,omitempty"`
	ChangePassword ChangePasswordPayload `json:"change_password,omitempty"`
	Email          EmailPayload          `json:"email,omitempty"`
	ID             IDPayload             `json:"id,omitempty"`
	Log            LogPayload            `json:"log,omitempty"`
	Mail           MailPayload           `json:"mail,omitempty"`
}

// MailPayload stores data to send mail to user
type MailPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

// AuthUserPayload stores data to authenticate user
type AuthUserPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RegUserPayload stores data to registration user
type RegUserPayload struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Password  string `json:"password"`
	Active    int    `json:"active"`
}

// UpdateUserPayload stores data to update user
type UpdateUserPayload struct {
	Email       string `json:"email"`
	EmailChange string `json:"email_change"`
	FirstName   string `json:"first_name,omitempty,omitempty"`
	LastName    string `json:"last_name,omitempty,omitempty"`
	Active      int    `json:"active,omitempty"`
}

// ChangePasswordPayload stores data to change password
type ChangePasswordPayload struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	NewPassword string `json:"new_password"`
}

// EmailPayload stores data of email
type EmailPayload struct {
	Email string `json:"email"`
}

// IDPayload stores id data
type IDPayload struct {
	ID int `json:"id"`
}

// LogPayload stores log data
type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

// Broker returns simple message of json
func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "Hit the broker",
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// HandleSubmission is basic function to check actions and do requests, return responses
func (app *Config) HandleSubmission(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	switch requestPayload.Action {
	case "authenticate_user":
		app.authenticateUserViaRabbit(w, requestPayload.Auth)
	case "registration_user":
		app.registrationUserViaRabbit(w, requestPayload.Reg)
	case "update_user":
		app.updateUser(w, requestPayload.UpdateUser)
	case "change_password_user":
		app.changePassword(w, requestPayload.ChangePassword)
	case "get_all_users":
		app.getAllUsers(w)
	case "get_user_by_email":
		app.getUserByEmailViaRabbit(w, requestPayload.Email)
	case "get_user_by_id":
		app.getUserByIDViaRabbit(w, requestPayload.ID)
	case "delete_user_by_email":
		app.deleteUserByEmail(w, requestPayload.Email)
	case "delete_user_by_id":
		app.deleteUserByID(w, requestPayload.ID)
	case "log":
		app.logEventViaRabbit(w, requestPayload.Log)
	case "mail":
		app.sendMailViaRabbit(w, requestPayload.Mail)

	default:
		app.errorJSON(w, errors.New("unknown action"))
	}
}

// authenticateUser auths user with email and password
func (app *Config) authenticateUser(w http.ResponseWriter, a AuthUserPayload) {
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

// updateUser updates user`s fields
func (app *Config) updateUser(w http.ResponseWriter, u UpdateUserPayload) {
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

// changePassword changes user`s password
func (app *Config) changePassword(w http.ResponseWriter, cp ChangePasswordPayload) {
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

// getAllUsers returns all users
func (app *Config) getAllUsers(w http.ResponseWriter) {
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

// getUserByEmail returns user by email
func (app *Config) getUserByEmail(w http.ResponseWriter, e EmailPayload) {
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

// getUserByID returns user by ID
func (app *Config) getUserByID(w http.ResponseWriter, i IDPayload) {
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

// deleteUserByEmail deletes user by email
func (app *Config) deleteUserByEmail(w http.ResponseWriter, e EmailPayload) {
	// create some json we'll send to the auth microservice
	jsonData, _ := json.MarshalIndent(e, "", "\t")

	// call the service
	request, err := http.NewRequest("DELETE", "http://authentication-service/delete_by_email", bytes.NewBuffer(jsonData))
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

// deleteUserByID deletes user by ID
func (app *Config) deleteUserByID(w http.ResponseWriter, i IDPayload) {
	// create some json we'll send to the auth microservice
	jsonData, _ := json.MarshalIndent(i, "", "\t")

	// call the service
	request, err := http.NewRequest("DELETE", "http://authentication-service/delete_by_id", bytes.NewBuffer(jsonData))
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

// registrationUser creates user and returns user`s ID
func (app *Config) registrationUser(w http.ResponseWriter, r RegUserPayload) {
	// create some json we'll send to the auth microservice
	jsonData, _ := json.MarshalIndent(r, "", "\t")

	// call the service
	request, err := http.NewRequest("POST", "http://authentication-service/registration", bytes.NewBuffer(jsonData))
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

// logItem requests of logger-service to log event
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

// sendMail sends message to user`s email
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

// sendMailViaRabbit sends async message to user`s email via RabbitMQ
func (app *Config) sendMailViaRabbit(w http.ResponseWriter, msg MailPayload) {
	var requestPayload RequestPayload
	requestPayload.Action = "mail"
	requestPayload.Mail = msg

	_, err := app.pushToQueue(requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = fmt.Sprintf("Message sent to %s", requestPayload.Mail.To)

	app.writeJSON(w, http.StatusAccepted, payload)
}

// logEventViaRabbit cals the async logger microservice via RabbitMQ
func (app *Config) logEventViaRabbit(w http.ResponseWriter, l LogPayload) {
	var requestPayload RequestPayload
	requestPayload.Action = "log"
	requestPayload.Log = l

	_, err := app.pushToQueue(requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "logged via RabbitMQ"

	app.writeJSON(w, http.StatusAccepted, payload)
}

// authenticateUserViaRabbit auths user with email and password via RabbitMQ
func (app *Config) authenticateUserViaRabbit(w http.ResponseWriter, a AuthUserPayload) {
	var requestPayload RequestPayload

	requestPayload.Action = "authenticate_user"
	requestPayload.Auth = a

	response, err := app.pushToQueue(requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	var payload jsonResponse

	err = json.Unmarshal(response, &payload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// getUserByEmailViaRabbit returns user by email via RabbitMQ
func (app *Config) getUserByEmailViaRabbit(w http.ResponseWriter, e EmailPayload) {
	var requestPayload RequestPayload

	requestPayload.Action = "get_user_by_email"
	requestPayload.Email = e

	response, err := app.pushToQueue(requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	var payload jsonResponse

	err = json.Unmarshal(response, &payload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// getUserByIDViaRabbit returns user by ID via RabbitMQ
func (app *Config) getUserByIDViaRabbit(w http.ResponseWriter, i IDPayload) {
	var requestPayload RequestPayload

	requestPayload.Action = "get_user_by_id"
	requestPayload.ID = i

	response, err := app.pushToQueue(requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	var payload jsonResponse

	err = json.Unmarshal(response, &payload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// registrationUserViaRabbit creates user and returns user`s ID via RabbitMQ
func (app *Config) registrationUserViaRabbit(w http.ResponseWriter, r RegUserPayload) {
	var requestPayload RequestPayload

	requestPayload.Action = "registration_user"
	requestPayload.Reg = r

	response, err := app.pushToQueue(requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	var payload jsonResponse

	err = json.Unmarshal(response, &payload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// pushToQueue pushes request to queue of RabbitMQ
func (app *Config) pushToQueue(payload RequestPayload) ([]byte, error) {
	var response []byte

	emitter, err := event.NewEventEmitter(payload.Action, app.Rabbit)
	if err != nil {
		return nil, err
	}

	var j []byte

	j, _ = json.MarshalIndent(&payload, "", "\t")

	switch payload.Action {
	case "log":
		err = emitter.Push(string(j), payload.Action, "log")
	case "mail":
		err = emitter.Push(string(j), payload.Action, "mail")
	case "authenticate_user":
		response, err = emitter.PushWithResponse(string(j), payload.Action, "authenticate.user")
	case "get_user_by_email":
		response, err = emitter.PushWithResponse(string(j), payload.Action, "get.user.by.email")
	case "get_user_by_id":
		response, err = emitter.PushWithResponse(string(j), payload.Action, "get.user.by.id")
	case "registration_user":
		response, err = emitter.PushWithResponse(string(j), payload.Action, "registration.user")
	default:
		log.Printf("invalid name of channel RabbitMQ %s", payload.Action)
	}

	if err != nil {
		return nil, err
	}

	return response, err
}

// RPCPayload stores log data RPC
type RPCPayload struct {
	Name string
	Data string
}

// logItemViaRpc logs some data via RPC
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

// LogViaGRPC logs some data via gRPC
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

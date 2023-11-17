package main

import (
	"authentication/data"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

func (app *Config) GetByEmail(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Email string `json:"email"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	// get user form database
	user, err := app.Models.User.GetByEmail(requestPayload.Email)
	if err != nil {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusBadRequest)
		return
	}

	// log getByEmail
	err = app.logRequest("receive user", fmt.Sprintf("%s received", user.Email))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("recived user"),
		Data:    user,
	}

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) Registrate(w http.ResponseWriter, r *http.Request) {
	var requestPayload data.User

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	// insert user in database
	id, err := app.Models.User.Insert(requestPayload)
	if err != nil {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusBadRequest)
		return
	}

	// log registrate
	err = app.logRequest("registrated", fmt.Sprintf("%s registrated in", requestPayload.Email))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Created user with id %s", id),
		Data:    id,
	}

	app.writeJSON(w, http.StatusCreated, payload)
}

func (app *Config) GetAll(w http.ResponseWriter, r *http.Request) {
	// get all users from database
	users, err := app.Models.User.GetAll()
	if err != nil {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusBadRequest)
		return
	}

	// log getAll
	err = app.logRequest("receive users", fmt.Sprintf("received %v users", len(users)))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Received %v users", len(users)),
		Data:    users,
	}

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) Authenticate(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	log.Println("authpayload", requestPayload)

	// validate the user against the database
	user, err := app.Models.User.GetByEmail(requestPayload.Email)
	if err != nil {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusBadRequest)
		return
	}

	valid, err := user.PasswordMatches(requestPayload.Password)
	if err != nil || !valid {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusBadRequest)
		return
	}

	// log authentication
	err = app.logRequest("authentication", fmt.Sprintf("%s logged in", user.Email))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Logged in user %s", user.Email),
		Data:    user,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}

func (app *Config) GetByID(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		ID int `json:"id"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	// validate the user against the database
	user, err := app.Models.User.GetOne(requestPayload.ID)
	if err != nil {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusBadRequest)
		return
	}

	// log getByID
	err = app.logRequest("receive user", fmt.Sprintf("%s received", user.Email))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("received user"),
		Data:    user,
	}

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) Update(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Email       string `json:"email"`
		EmailChange string `json:"email_change"`
		FirstName   string `json:"first_name,omitempty,omitempty"`
		LastName    string `json:"last_name,omitempty,omitempty"`
		Active      int    `json:"active,omitempty"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	// get user from database
	user, err := app.Models.User.GetByEmail(requestPayload.Email)
	if err != nil {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusBadRequest)
		return
	}

	// check updated objects
	if requestPayload.EmailChange != "" {
		user.Email = requestPayload.EmailChange
	}
	if requestPayload.Active != user.Active {
		user.Active = requestPayload.Active
	}
	if requestPayload.FirstName != "" {
		user.FirstName = requestPayload.FirstName
	}
	if requestPayload.LastName != "" {
		user.LastName = requestPayload.LastName
	}

	// update user
	err = user.Update()
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// log update
	err = app.logRequest("update", fmt.Sprintf("%s updated, now %s", requestPayload.Email, user.Email))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Updated user with id %s", user.ID),
		Data:    user.Email,
	}

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		NewPassword string `json:"new_password"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	// get user from database
	user, err := app.Models.User.GetByEmail(requestPayload.Email)
	if err != nil {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusBadRequest)
		return
	}

	log.Println("user", user)
	log.Println("requestPayload", requestPayload.Password)

	// check user`s password
	valid, err := user.PasswordMatches(requestPayload.Password)
	if err != nil || !valid {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusBadRequest)
		return
	}

	log.Println("valid", valid)

	// update user`s password
	err = user.ResetPassword(requestPayload.NewPassword)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// log update
	err = app.logRequest("change password", fmt.Sprintf("%s changed password", requestPayload.Email))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Changed user`s password with id %v", user.ID),
		Data:    user.Email,
	}

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) GetByEmailDelete(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Email string `json:"email"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	// get user form database
	user, err := app.Models.User.GetByEmail(requestPayload.Email)
	if err != nil {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusBadRequest)
		return
	}

	err = user.Delete()
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// log getByEmail
	err = app.logRequest("delete user", fmt.Sprintf("%s deleted", requestPayload.Email))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("user deleted"),
		Data:    "",
	}

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) GetByIDDelete(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		ID int `json:"id"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	// validate the user against the database
	err = app.Models.User.DeleteByID(requestPayload.ID)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// log getByID
	err = app.logRequest("delete user", fmt.Sprintf("user with id %s deleted", requestPayload.ID))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("deleted user"),
		Data:    "",
	}

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) logRequest(name, data string) error {
	var entry struct {
		Name string `json:"name"`
		Data string `json:"data"`
	}

	entry.Name = name
	entry.Data = data

	jsonData, _ := json.MarshalIndent(entry, "", "\t")
	logServiceURL := "http://logger-service/log"

	request, err := http.NewRequest("POST", logServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	client := &http.Client{}
	_, err = client.Do(request)
	if err != nil {
		return err
	}

	return nil
}

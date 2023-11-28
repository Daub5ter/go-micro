package main

import (
	"authentication/data"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

// GetByEmail returns user by email
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
	go app.logRequest("receive user", fmt.Sprintf("%s received", user.Email))

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("recived user"),
		Data:    user,
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// Registration creates user and returns user`s ID
func (app *Config) Registration(w http.ResponseWriter, r *http.Request) {
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

	// log registration
	go app.logRequest("registered", fmt.Sprintf("%s registered in", requestPayload.Email))

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Created user with id %v", id),
		Data:    id,
	}

	app.writeJSON(w, http.StatusCreated, payload)
}

// GetAll returns all users
func (app *Config) GetAll(w http.ResponseWriter, r *http.Request) {
	// get all users from database
	users, err := app.Models.User.GetAll()
	if err != nil {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusBadRequest)
		return
	}

	// log getAll
	go app.logRequest("receive users", fmt.Sprintf("received %v users", len(users)))

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Received %v users", len(users)),
		Data:    users,
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// Authenticate auths user with email and password
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

	// validate the user against the database
	user, err := app.Models.User.GetByEmailWithPassword(requestPayload.Email)
	if err != nil {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusBadRequest)
		return
	}

	valid, err := user.PasswordMatches(requestPayload.Password)
	if err != nil || !valid {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusBadRequest)
		return
	}

	jwtToken, err := app.Models.UserJWT.CreateJWTToken(requestPayload.Email)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	// log authentication
	go app.logRequest("authentication", fmt.Sprintf("%s logged in", user.Email))

	// structure for response without password
	u := struct {
		ID           int       `json:"id"`
		Email        string    `json:"email"`
		FirstName    string    `json:"first_name,omitempty"`
		LastName     string    `json:"last_name,omitempty"`
		Active       int       `json:"active"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		SessionToken string    `json:"session_token"`
	}{}

	u.ID = user.ID
	u.Email = user.Email
	u.FirstName = user.FirstName
	u.LastName = user.LastName
	u.Active = user.Active
	u.CreatedAt = user.CreatedAt
	u.UpdatedAt = user.UpdatedAt
	u.SessionToken = jwtToken

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Logged in user %s", u.Email),
		Data:    u,
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// GetByID returns user by ID
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
	go app.logRequest("receive user", fmt.Sprintf("%s received", user.Email))

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("received user"),
		Data:    user,
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// Update updates user`s fields
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
	go app.logRequest("update", fmt.Sprintf("%s updated, now %s", requestPayload.Email, user.Email))

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Updated user with id %v", user.ID),
		Data:    user.Email,
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// ChangePassword changes user`s password
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
	user, err := app.Models.User.GetByEmailWithPassword(requestPayload.Email)
	if err != nil {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusBadRequest)
		return
	}

	// check user`s password
	valid, err := user.PasswordMatches(requestPayload.Password)
	if err != nil || !valid {
		app.errorJSON(w, errors.New("invalid credentials"), http.StatusBadRequest)
		return
	}

	// update user`s password
	err = user.ResetPassword(requestPayload.NewPassword)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// log update
	go app.logRequest("change password", fmt.Sprintf("%s changed password", requestPayload.Email))

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Changed user`s password with id %v", user.ID),
		Data:    user.Email,
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// DeleteByEmail deletes user by email
func (app *Config) DeleteByEmail(w http.ResponseWriter, r *http.Request) {
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
	go app.logRequest("delete user", fmt.Sprintf("%s deleted", requestPayload.Email))

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("user deleted"),
		Data:    "",
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// DeleteByID deletes user by ID
func (app *Config) DeleteByID(w http.ResponseWriter, r *http.Request) {
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
	go app.logRequest("delete user", fmt.Sprintf("user with id %v deleted", requestPayload.ID))

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("deleted user"),
		Data:    "",
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// AuthenticateSession checks valid or not session of user
func (app *Config) AuthenticateSession(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		SessionToken string `json:"session_token"`
	}

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	email, err := app.Models.UserJWT.CheckJWTToken(requestPayload.SessionToken)
	if err != nil {
		app.errorJSON(w, err, http.StatusBadRequest)
		return
	}

	// log getByEmail
	go app.logRequest("checked users session", fmt.Sprintf("%s`s session is valid", email))

	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("session is valid"),
		Data:    email,
	}

	app.writeJSON(w, http.StatusOK, payload)
}

// logRequest requests of logger-service to log event
func (app *Config) logRequest(name, data string) {
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
		log.Println(err)
	}

	client := &http.Client{}
	_, err = client.Do(request)
	if err != nil {
		log.Println(err)
	}
}

package main

import (
	"analysis/data"
	"net/http"
)

type JSONPayload struct {
	Email string `json:"email"`
}

func (app *Config) WriteAnalysis(w http.ResponseWriter, r *http.Request) {
	// read json into var
	var requestPayload JSONPayload
	app.readJSON(w, r, &requestPayload)

	// insert data
	event := data.ActionsUser{
		Email: requestPayload.Email,
	}

	err := app.Models.ActionsUser.Set(event)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	resp := jsonResponse{
		Error:   false,
		Message: "set",
	}

	app.writeJSON(w, http.StatusAccepted, resp)
}

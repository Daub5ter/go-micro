package main

import (
	"analysis/data"
	"log"
	"net/http"
	"time"
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

	value, err := app.Models.ActionsUser.Get(event)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	value++
	event.Actions = value

	err = app.Models.ActionsUser.Set(event)
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

func (app *Config) UpdateDB(done chan bool) {
	log.Println("Starting updating db...")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			keyValues, keys, err := app.Models.ActionsUser.GetAll()
			if err != nil {
				log.Println("Error to get keys + values while update DB", err)
			}

			err = app.Models.ActionsUser.DeleteSome(keys)
			if err != nil {
				log.Println("Error to delete keys while update DB", err)
			}

			for key, val := range keyValues {
				log.Println("key, val", key, val)
			}

			log.Println("Analysis DB updated")
			done <- true
			return
		}
	}
}

package main

import (
	"analysis/data"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
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
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			log.Println("Starting updating db...")

			keyValues, keys, err := app.Models.ActionsUser.GetAll()
			if err != nil {
				log.Println("Error to get keys + values while update DB", err)
			}

			if len(keys) != 0 {
				err = app.Models.ActionsUser.DeleteSome(keys)
				if err != nil {
					log.Println("Error to delete keys while update DB", err)
				}
			}

			var totalActions int

			analUsers, err := app.Models.AnalysisUser.GetAll()
			if err != nil {
				log.Printf("error with get all %v", err)
			}

			for _, au := range analUsers {
				totalActions += au.Actions
			}

			for _, val := range keyValues {
				value, err := strconv.Atoi(val)
				if err != nil {
					log.Println(err)
				}
				totalActions += value
			}

			for _, au := range analUsers {
				v, ok := keyValues[fmt.Sprintf("actions %s", au.Email)]
				if ok {
					actions, err := strconv.Atoi(v)
					if err != nil {
						log.Println(err)
					}
					au.PercentActivity = 100 * float64(au.Actions+actions) / float64(totalActions)
				} else {
					au.PercentActivity = 100 * float64(au.Actions) / float64(totalActions)
				}

				_, err = au.Update()
				if err != nil {
					log.Printf("error with update: %v", err)
				}
			}

			for key, val := range keyValues {
				key = strings.ReplaceAll(key, "actions ", "")
				analUser, err := app.Models.AnalysisUser.GetOneByEmail(key)
				if err != nil {
					log.Println(err)
				}

				if analUser == nil {
					actions, err := strconv.Atoi(val)
					if err != nil {
						log.Println(err)
					}

					err = app.Models.AnalysisUser.Insert(
						data.AnalysisUser{
							Email:           key,
							Actions:         actions,
							PercentActivity: 100 * float64(actions) / float64(totalActions),
						})
					if err != nil {
						log.Printf("error with insert: %v", err)
					}
				}
			}

			log.Println("Analysis db updated. Next update will after 30 seconds")

			done <- true
			return
		}
	}
}

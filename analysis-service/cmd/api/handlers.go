package main

import (
	"analysis/data"
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

			for key, val := range keyValues {
				key = strings.ReplaceAll(key, "actions ", "")
				analUser, err := app.Models.AnalysisUser.GetOneByEmail(key)
				if err != nil {
					log.Println(err)
				}

				actions, err := strconv.Atoi(val)
				if err != nil {
					log.Println(err)
				}

				if analUser == nil {
					err = app.Models.AnalysisUser.Insert(
						data.AnalysisUser{
							Email:           key,
							Actions:         actions,
							PercentActivity: float64(100 * actions / totalActions),
						})
					if err != nil {
						log.Printf("error with insert: %v", err)
					}
				} else {
					analUser.Actions = analUser.Actions + actions
					analUser.PercentActivity = float64(100 * (analUser.Actions + actions) / totalActions)

					_, err = analUser.Update()
					if err != nil {
						log.Printf("error with update: %v", err)
					}
				}
			}

			for _, au := range analUsers {
				au.PercentActivity = float64(100 * (au.Actions) / totalActions)

				_, err = au.Update()
				if err != nil {
					log.Printf("error with update: %v", err)
				}
			}

			log.Println("Analysis db updated. Next update will after 30 seconds")

			done <- true
		}
	}
}

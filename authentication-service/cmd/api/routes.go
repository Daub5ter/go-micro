package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (app *Config) routes() http.Handler {
	mux := chi.NewRouter()

	// specify who is allowed to connect
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	mux.Use(middleware.Heartbeat("/ping"))

	mux.Get("/get_all", app.GetAll)
	mux.Put("/update", app.Update)
	mux.Put("/change_password", app.ChangePassword)
	mux.Post("/authenticate", app.Authenticate)
	mux.Post("/registrate", app.Registrate)
	mux.Post("/get_by_email", app.GetByEmail)
	mux.Post("/get_by_id", app.GetByID)
	mux.Delete("/get_by_email_delete", app.GetByEmailDelete)
	mux.Delete("/get_by_id_delete", app.GetByIDDelete)

	return mux
}

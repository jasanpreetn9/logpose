package api

import (
	"github.com/go-chi/cors"
)

var CORS = cors.Handler(cors.Options{
	AllowedOrigins:   []string{"http://localhost:*", "http://127.0.0.1:*"},
	AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
	ExposedHeaders:   []string{"Link"},
	AllowCredentials: false,
	MaxAge:           300,
})

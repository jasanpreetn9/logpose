package api

import (
	"encoding/json"
	"net/http"

	"onepace-library/internal/library"
)

func HandleGetLibrary(store *library.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		store.Read(func(lib *library.Library) {
			json.NewEncoder(w).Encode(lib)
		})
	}
}

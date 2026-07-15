package api

import (
	"encoding/json"
	"net/http"

	"onepace-library/internal/version"
)

func HandleGetVersion() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"version": version.Version})
	}
}

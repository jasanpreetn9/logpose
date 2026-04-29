package api

import (
	"encoding/json"
	"net/http"

	"onepace-library/internal/metadata"

	"github.com/go-chi/chi/v5"
)

func HandleGetEpisode(meta *metadata.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		crc := chi.URLParam(r, "crc")

		ep, err := meta.GetEpisodeByCRC32(crc)
		if err != nil {
			http.Error(w, "episode not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ep)
	}
}

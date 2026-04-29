package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
)

type MonitorRequest struct {
	Arc       int  `json:"arc"`
	Episode   int  `json:"episode"`
	Monitored bool `json:"monitored"`
}

func HandleMonitorEpisode(meta *metadata.Client, store *library.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req MonitorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		arcTitle := meta.GetArcTitle(req.Arc)
		key := fmt.Sprintf("%d", req.Episode)

		if err := store.Write(func(lib *library.Library) error {
			arc := lib.GetOrCreateArc(req.Arc, arcTitle)
			ep, ok := arc.Episodes[key]
			if !ok {
				ep = library.Episode{EpisodeNumber: req.Episode}
			}
			ep.Monitored = req.Monitored
			arc.Episodes[key] = ep
			return nil
		}); err != nil {
			http.Error(w, "failed to save library", http.StatusInternalServerError)
			return
		}

		status := "monitored"
		if !req.Monitored {
			status = "unmonitored"
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": status})
	}
}

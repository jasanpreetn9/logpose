package api

import (
	"encoding/json"
	"net/http"

	"onepace-library/internal/activity"
)

func HandleGetActivity(acts *activity.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		events := acts.All()
		if events == nil {
			events = []activity.Event{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(events)
	}
}

func HandleGetHistory(acts *activity.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		events := acts.ByType(activity.EventImport)
		if events == nil {
			events = []activity.Event{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(events)
	}
}

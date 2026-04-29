package api

import (
	"encoding/json"
	"net/http"

	"onepace-library/internal/activity"
	"onepace-library/internal/metadata"
	"onepace-library/internal/qbittorrent"
)

type AddDownloadRequest struct {
	CRC32 string `json:"crc32"`
}

func HandleAddToQbit(meta *metadata.Client, qb *qbittorrent.Client, acts *activity.Store, enabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !enabled {
			http.Error(w, "qBittorrent is not enabled — configure it in Settings", http.StatusServiceUnavailable)
			return
		}

		var req AddDownloadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		ep, err := meta.GetEpisodeByCRC32(req.CRC32)
		if err != nil {
			http.Error(w, "episode not found", http.StatusNotFound)
			return
		}

		if ep.File.URL == "" {
			http.Error(w, "no download URL available", http.StatusInternalServerError)
			return
		}

		if err := qb.AddTorrent(ep.File.URL); err != nil {
			acts.Add(activity.EventDownloadFailed, "Download failed: "+ep.Title, err.Error(), false)
			http.Error(w, "qBit add failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		acts.Add(activity.EventDownloadQueued, "Download queued: "+ep.Title, ep.File.URL, true)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"message": "download added to qBittorrent",
			"url":     ep.File.URL,
		})
	}
}

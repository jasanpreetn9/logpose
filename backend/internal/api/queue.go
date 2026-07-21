package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"onepace-library/internal/metadata"
	"onepace-library/internal/qbittorrent"
	"onepace-library/internal/scanner"
)

type QueueItem struct {
	Hash     string  `json:"hash"`
	Name     string  `json:"name"`
	Title    string  `json:"title"`
	Arc      int     `json:"arc"`
	Episode  int     `json:"episode"`
	Progress float64 `json:"progress"`
	Size     int64   `json:"size"`
	DLSpeed  int64   `json:"dlspeed"`
	ETA      int64   `json:"eta"`
	State    string  `json:"state"`
}

// HandleGetQueue returns all torrents in the Logpose category, enriched with
// episode info where the torrent name parses.
// GET /api/queue
func HandleGetQueue(meta *metadata.Client, qb *qbittorrent.Client, enabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items := []QueueItem{}
		if enabled {
			torrents, err := qb.GetTorrents()
			if err != nil {
				http.Error(w, "qBittorrent: "+err.Error(), http.StatusBadGateway)
				return
			}
			for _, t := range torrents {
				item := QueueItem{
					Hash:     t.Hash,
					Name:     t.Name,
					Progress: t.Progress,
					Size:     t.Size,
					DLSpeed:  t.DLSpeed,
					ETA:      t.ETA,
					State:    t.State,
				}
				if parsed, err := scanner.ParseOnePaceFilename(t.Name); err == nil {
					if epMeta, err := meta.GetEpisodeByCRC32(parsed.CRC32); err == nil {
						item.Title = epMeta.Title
						item.Arc = epMeta.Arc
						item.Episode = epMeta.Episode
					}
				}
				items = append(items, item)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	}
}

// HandleDeleteQueueItem removes a torrent from qBittorrent (keeps files).
// DELETE /api/queue/{hash}
func HandleDeleteQueueItem(qb *qbittorrent.Client, enabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !enabled {
			http.Error(w, "qBittorrent is not enabled", http.StatusServiceUnavailable)
			return
		}
		hash := chi.URLParam(r, "hash")
		if hash == "" {
			http.Error(w, "missing hash", http.StatusBadRequest)
			return
		}
		if err := qb.DeleteTorrent(hash); err != nil {
			http.Error(w, "delete failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

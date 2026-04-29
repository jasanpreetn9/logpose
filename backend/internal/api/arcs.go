package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"onepace-library/internal/activity"
	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
	"onepace-library/internal/nfo"
	"onepace-library/internal/qbittorrent"
)

// HandleMonitorArc sets the monitored flag on every episode in the arc (and the arc itself).
// It creates library entries for any metadata episodes that don't yet exist.
// POST /api/arcs/{arcId}/monitor   body: {"monitored": bool}
func HandleMonitorArc(meta *metadata.Client, store *library.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		arcId, err := parseArcId(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var req struct {
			Monitored bool `json:"monitored"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		arcTitle := meta.GetArcTitle(arcId)
		metaEps := meta.EpisodesByArc(arcId)
		var updated int

		if err := store.Write(func(lib *library.Library) error {
			arc := lib.GetOrCreateArc(arcId, arcTitle)
			arc.Monitored = req.Monitored

			// Update existing episodes.
			for key, ep := range arc.Episodes {
				ep.Monitored = req.Monitored
				arc.Episodes[key] = ep
			}

			// Create entries for metadata episodes not yet in the library.
			for _, epMeta := range metaEps {
				key := fmt.Sprintf("%d", epMeta.Episode)
				if _, exists := arc.Episodes[key]; exists {
					updated++
					continue
				}
				arc.Episodes[key] = library.Episode{
					EpisodeNumber:  epMeta.Episode,
					CRC32:          epMeta.File.CRC32,
					Title:          epMeta.Title,
					Description:    epMeta.Description,
					DownloadStatus: "missing",
					Monitored:      req.Monitored,
				}
				updated++
			}
			return nil
		}); err != nil {
			http.Error(w, "failed to save", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"updated": updated,
		})
	}
}

// HandleDownloadMonitored queues all monitored+missing episodes in the arc to qBittorrent.
// POST /api/arcs/{arcId}/download-monitored
func HandleDownloadMonitored(meta *metadata.Client, store *library.Store, qb *qbittorrent.Client, acts *activity.Store, enabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !enabled {
			http.Error(w, "qBittorrent is not enabled — configure it in Settings", http.StatusServiceUnavailable)
			return
		}

		arcId, err := parseArcId(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		type toQueue struct {
			crc32 string
			title string
			url   string
		}
		var candidates []toQueue

		// Use metadata as the authoritative list of episodes for this arc.
		// The library is only consulted for per-episode monitored/status overrides.
		metaEps := meta.EpisodesByArc(arcId)

		store.Read(func(lib *library.Library) {
			arc := lib.Arcs[arcId] // may be nil for a fresh arc
			arcMonitored := arc != nil && arc.Monitored

			for _, epMeta := range metaEps {
				if epMeta.File.URL == "" {
					continue
				}

				key := fmt.Sprintf("%d", epMeta.Episode)
				isMonitored := arcMonitored
				isImported := false

				if arc != nil {
					if libEp, ok := arc.Episodes[key]; ok {
						isMonitored = libEp.Monitored
						isImported = libEp.DownloadStatus == "imported"
					}
				}

				if !isMonitored || isImported {
					continue
				}

				candidates = append(candidates, toQueue{
					crc32: epMeta.File.CRC32,
					title: epMeta.Title,
					url:   epMeta.File.URL,
				})
			}
		})

		queued := 0
		for _, c := range candidates {
			if err := qb.AddTorrent(c.url); err != nil {
				acts.Add(activity.EventDownloadFailed, "Download failed: "+c.title, err.Error(), false)
				log.Printf("download-monitored: failed to queue %s: %v", c.crc32, err)
				continue
			}
			acts.Add(activity.EventDownloadQueued, "Download queued: "+c.title, c.url, true)
			queued++
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"queued": queued,
			"total":  len(candidates),
		})
	}
}

// HandleVerifyNFOs regenerates NFO files for all imported episodes in the arc.
// POST /api/arcs/{arcId}/verify-nfo
func HandleVerifyNFOs(meta *metadata.Client, store *library.Store, acts *activity.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		arcId, err := parseArcId(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		type toRegen struct {
			ep       library.Episode
			arcTitle string
		}
		var work []toRegen

		store.Read(func(lib *library.Library) {
			arc, ok := lib.Arcs[arcId]
			if !ok {
				return
			}
			arcTitle := meta.GetArcTitle(arcId)
			for _, ep := range arc.Episodes {
				if ep.FilePath == "" {
					continue
				}
				work = append(work, toRegen{ep: ep, arcTitle: arcTitle})
			}
		})

		updated := 0
		for _, w := range work {
			epMeta, err := meta.GetEpisodeByCRC32(w.ep.CRC32)
			if err != nil {
				log.Printf("verify-nfo: metadata not found for CRC %s: %v", w.ep.CRC32, err)
				continue
			}
			nfoPath := nfo.NFOPathForVideo(w.ep.FilePath)
			if err := nfo.GenerateEpisodeNFO(w.ep, epMeta, w.arcTitle, nfoPath); err != nil {
				log.Printf("verify-nfo: failed to write %s: %v", nfoPath, err)
				continue
			}
			updated++
		}

		acts.Add(activity.EventLibraryScan,
			fmt.Sprintf("NFO verify complete — %d/%d updated", updated, len(work)),
			"",
			true,
		)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"updated": updated,
			"total":   len(work),
		})
	}
}

func parseArcId(r *http.Request) (int, error) {
	s := chi.URLParam(r, "arcId")
	id, err := strconv.Atoi(s)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid arcId: %q", s)
	}
	return id, nil
}

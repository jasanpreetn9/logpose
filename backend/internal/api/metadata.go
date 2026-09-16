package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"onepace-library/internal/activity"
	"onepace-library/internal/config"
	"onepace-library/internal/downloads"
	"onepace-library/internal/grabber"
	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
	"onepace-library/internal/nfo"
	"onepace-library/internal/qbittorrent"
)

// HandleRefreshMetadata re-fetches episode and arc metadata from the source
// URLs and regenerates NFOs for any episodes whose metadata changed.
// POST /api/metadata/refresh
func HandleRefreshMetadata(meta *metadata.Client, cfg *config.Config, store *library.Store, acts *activity.Store, qb *qbittorrent.Client, tracker *downloads.Tracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Pick up any URL changes saved in Settings since startup.
		meta.EpisodesURL = cfg.Metadata.EpisodesURL
		meta.ArcsURL = cfg.Metadata.ArcsURL

		if err := meta.Refresh(); err != nil {
			acts.Add(activity.EventMetadataRefresh, "Metadata refresh failed", err.Error(), false)
			http.Error(w, "metadata refresh failed: "+err.Error(), http.StatusBadGateway)
			return
		}

		nfosUpdated := RegenerateStaleNFOs(meta, store)
		backfilled := SyncMonitoredEpisodes(meta, store)
		episodes, arcs := meta.Counts()

		acts.Add(activity.EventMetadataRefresh,
			"Metadata refreshed",
			fmt.Sprintf("%d episodes, %d arcs, %d NFOs regenerated", episodes, arcs, nfosUpdated),
			true,
		)
		if backfilled > 0 {
			log.Printf("Backfilled monitored flag for %d newly-appeared episode(s)", backfilled)
		}

		grabbed := 0
		if cfg.AutoDownload && cfg.QBittorrent.Enabled {
			grabbed = grabber.GrabWanted(meta, store, qb, acts, tracker)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":      "ok",
			"episodes":    episodes,
			"arcs":        arcs,
			"nfosUpdated": nfosUpdated,
			"grabbed":     grabbed,
			"lastUpdated": meta.LastUpdated(),
		})
	}
}

// RegenerateStaleNFOs re-generates NFO files for any imported episodes whose
// metadata changed since the last refresh. Returns the number regenerated.
func RegenerateStaleNFOs(meta *metadata.Client, store *library.Store) int {
	stale := meta.StaleEpisodes()
	if len(stale) == 0 {
		return 0
	}
	updated := 0
	store.Read(func(lib *library.Library) {
		for _, arc := range lib.Arcs {
			for _, ep := range arc.Episodes {
				for _, v := range ep.Versions {
					for _, crc := range stale {
						if v.CRC32 == crc && v.FilePath != "" {
							epMeta, err := meta.GetEpisodeByCRC32(crc)
							if err != nil {
								continue
							}
							arcTitle := meta.GetArcTitle(arc.ArcNumber)
							nfoPath := nfo.NFOPathForVideo(v.FilePath)
							if err := nfo.GenerateEpisodeNFO(ep, v, epMeta, arcTitle, nfoPath); err != nil {
								log.Printf("Failed to regenerate NFO for %s (CRC %s): %v", ep.Title, crc, err)
								continue
							}
							log.Printf("Regenerated NFO for %s (CRC %s)", ep.Title, crc)
							updated++
						}
					}
				}
			}
		}
	})
	return updated
}

// SyncMonitoredEpisodes creates library entries for any metadata episodes
// that don't exist yet on an already-monitored arc, marking them monitored.
// Without this, an episode that appears in metadata after its arc was
// monitored (the arc didn't have that episode listed yet when Monitor Arc
// was clicked) would default to unmonitored and silently never surface on
// Wanted or get auto-grabbed. Returns the number of episodes backfilled.
func SyncMonitoredEpisodes(meta *metadata.Client, store *library.Store) int {
	backfilled := 0
	store.Write(func(lib *library.Library) error {
		for arcNumber, arc := range lib.Arcs {
			if !arc.Monitored {
				continue
			}
			for _, epMeta := range meta.EpisodesByArc(arcNumber) {
				key := fmt.Sprintf("%d", epMeta.Episode)
				if _, exists := arc.Episodes[key]; exists {
					continue
				}
				arc.Episodes[key] = library.Episode{
					EpisodeNumber: epMeta.Episode,
					Title:         epMeta.Title,
					Description:   epMeta.Description,
					Monitored:     true,
					Versions:      map[string]library.EpisodeVersion{},
				}
				backfilled++
			}
		}
		return nil
	})
	return backfilled
}

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
		episodes, arcs := meta.Counts()

		acts.Add(activity.EventMetadataRefresh,
			"Metadata refreshed",
			fmt.Sprintf("%d episodes, %d arcs, %d NFOs regenerated", episodes, arcs, nfosUpdated),
			true,
		)

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
				for _, crc := range stale {
					if ep.CRC32 == crc && ep.FilePath != "" {
						epMeta, err := meta.GetEpisodeByCRC32(crc)
						if err != nil {
							continue
						}
						arcTitle := meta.GetArcTitle(arc.ArcNumber)
						nfoPath := nfo.NFOPathForVideo(ep.FilePath)
						if err := nfo.GenerateEpisodeNFO(ep, epMeta, arcTitle, nfoPath); err != nil {
							log.Printf("Failed to regenerate NFO for %s (CRC %s): %v", ep.Title, crc, err)
							continue
						}
						log.Printf("Regenerated NFO for %s (CRC %s)", ep.Title, crc)
						updated++
					}
				}
			}
		}
	})
	return updated
}

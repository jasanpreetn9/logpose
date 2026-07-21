package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"onepace-library/internal/config"
	"onepace-library/internal/metadata"
	"onepace-library/internal/qbittorrent"
)

type HealthCheck struct {
	ID      string `json:"id"`
	Level   string `json:"level"` // "warning" or "error"
	Message string `json:"message"`
}

// HandleHealth reports issues Logpose can detect on its own — a dropped
// qBittorrent connection, metadata that stopped refreshing, a missing NAS
// mount, or a nonsensical config combination — so they surface in the UI
// instead of only appearing in server logs.
// GET /api/health
func HandleHealth(cfg *config.Config, meta *metadata.Client, qb *qbittorrent.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		checks := []HealthCheck{}

		if cfg.QBittorrent.Enabled {
			if _, err := qb.GetTorrents(); err != nil {
				checks = append(checks, HealthCheck{
					ID:      "qbittorrent",
					Level:   "error",
					Message: fmt.Sprintf("qBittorrent unreachable at %s: %v", cfg.QBittorrent.Host, err),
				})
			}
		}

		if age := time.Since(meta.LastUpdated()); age > 2*cfg.RefreshDuration() {
			checks = append(checks, HealthCheck{
				ID:    "metadata-stale",
				Level: "warning",
				Message: fmt.Sprintf(
					"Metadata hasn't refreshed in %s — check episodesUrl/arcsUrl or network access.",
					age.Round(time.Minute),
				),
			})
		}

		if cfg.AutoDownload && !cfg.QBittorrent.Enabled {
			checks = append(checks, HealthCheck{
				ID:      "auto-download-no-client",
				Level:   "warning",
				Message: "Auto Download is on but qBittorrent is disabled — nothing will be grabbed.",
			})
		}

		paths := []struct{ id, path string }{
			{"library-path", cfg.LibraryPath},
			{"download-path", cfg.DownloadPath},
		}
		for _, p := range paths {
			if _, err := os.Stat(p.path); err != nil {
				checks = append(checks, HealthCheck{
					ID:      p.id,
					Level:   "error",
					Message: fmt.Sprintf("%s is not accessible: %v", p.path, err),
				})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":     len(checks) == 0,
			"checks": checks,
		})
	}
}

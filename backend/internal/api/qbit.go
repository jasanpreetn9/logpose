package api

import (
	"encoding/json"
	"net/http"

	"onepace-library/internal/config"
	"onepace-library/internal/qbittorrent"
)

// HandleTestQBittorrent attempts a login against qBittorrent using the
// supplied credentials, falling back to the saved config for any blank field.
// Always responds 200 with {"ok": bool, ...} — a failed connection is a test
// result, not a server error.
// POST /api/qbittorrent/test   body: {"host","username","password"} (all optional)
func HandleTestQBittorrent(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Host     string `json:"host"`
			Username string `json:"username"`
			Password string `json:"password"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		if req.Host == "" {
			req.Host = cfg.QBittorrent.Host
		}
		if req.Username == "" {
			req.Username = cfg.QBittorrent.Username
		}
		if req.Password == "" {
			req.Password = cfg.QBittorrent.Password
		}

		w.Header().Set("Content-Type", "application/json")

		qb := qbittorrent.NewClient(req.Host, req.Username, req.Password)
		if err := qb.Login(); err != nil {
			json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
			return
		}

		version, err := qb.Version()
		if err != nil {
			// Login worked; version is best-effort.
			json.NewEncoder(w).Encode(map[string]any{"ok": true})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "version": version})
	}
}

package api

import (
	"encoding/json"
	"net/http"
	"time"

	"onepace-library/internal/config"
)

type ConfigResponse struct {
	Port            string `json:"port"`
	LibraryPath     string `json:"libraryPath"`
	DownloadPath    string `json:"downloadPath"`
	LibraryJSONPath string `json:"libraryJsonPath"`

	MetadataEpisodesURL     string `json:"metadataEpisodesUrl"`
	MetadataArcsURL         string `json:"metadataArcsUrl"`
	MetadataRefreshInterval string `json:"metadataRefreshInterval"`

	QBEnabled  bool   `json:"qbEnabled"`
	QBHost     string `json:"qbHost"`
	QBUsername string `json:"qbUsername"`
}

type ConfigUpdateRequest struct {
	Port            string `json:"port"`
	LibraryPath     string `json:"libraryPath"`
	DownloadPath    string `json:"downloadPath"`
	LibraryJSONPath string `json:"libraryJsonPath"`

	MetadataEpisodesURL     string `json:"metadataEpisodesUrl"`
	MetadataArcsURL         string `json:"metadataArcsUrl"`
	MetadataRefreshInterval string `json:"metadataRefreshInterval"`

	QBEnabled  bool   `json:"qbEnabled"`
	QBHost     string `json:"qbHost"`
	QBUsername string `json:"qbUsername"`
	QBPassword string `json:"qbPassword"`
}

func HandleGetConfig(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := ConfigResponse{
			Port:                    cfg.Port,
			LibraryPath:             cfg.LibraryPath,
			DownloadPath:            cfg.DownloadPath,
			LibraryJSONPath:         cfg.LibraryJSONPath,
			MetadataEpisodesURL:     cfg.Metadata.EpisodesURL,
			MetadataArcsURL:         cfg.Metadata.ArcsURL,
			MetadataRefreshInterval: cfg.MetadataRefreshInterval,
			QBEnabled:               cfg.QBittorrent.Enabled,
			QBHost:                  cfg.QBittorrent.Host,
			QBUsername:              cfg.QBittorrent.Username,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func HandleUpdateConfig(cfg *config.Config, cfgPath string, tickerReset chan<- time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ConfigUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		oldInterval := cfg.MetadataRefreshInterval

		if req.Port != "" {
			cfg.Port = req.Port
		}
		if req.LibraryPath != "" {
			cfg.LibraryPath = req.LibraryPath
		}
		if req.DownloadPath != "" {
			cfg.DownloadPath = req.DownloadPath
		}
		if req.LibraryJSONPath != "" {
			cfg.LibraryJSONPath = req.LibraryJSONPath
		}
		if req.MetadataEpisodesURL != "" {
			cfg.Metadata.EpisodesURL = req.MetadataEpisodesURL
		}
		if req.MetadataArcsURL != "" {
			cfg.Metadata.ArcsURL = req.MetadataArcsURL
		}
		if req.MetadataRefreshInterval != "" {
			cfg.MetadataRefreshInterval = req.MetadataRefreshInterval
		}
		cfg.QBittorrent.Enabled = req.QBEnabled
		if req.QBHost != "" {
			cfg.QBittorrent.Host = req.QBHost
		}
		if req.QBUsername != "" {
			cfg.QBittorrent.Username = req.QBUsername
		}
		if req.QBPassword != "" {
			cfg.QBittorrent.Password = req.QBPassword
		}

		// Validate before saving.
		if errs := cfg.Validate(); errs != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{"errors": errs})
			return
		}

		if err := config.Save(cfgPath, cfg); err != nil {
			http.Error(w, "failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Signal the metadata ticker if the interval changed and is valid.
		if cfg.MetadataRefreshInterval != oldInterval {
			if d, err := time.ParseDuration(cfg.MetadataRefreshInterval); err == nil {
				select {
				case tickerReset <- d:
				default:
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "config saved"})
	}
}

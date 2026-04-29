package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type MetadataConfig struct {
	EpisodesURL string `yaml:"episodesUrl"`
	ArcsURL     string `yaml:"arcsUrl"`
}

type Config struct {
	Port            string `yaml:"port"`
	LibraryPath     string `yaml:"libraryPath"`
	DownloadPath    string `yaml:"downloadPath"`
	LibraryJSONPath string `yaml:"libraryJsonPath"`

	MetadataRefreshInterval string `yaml:"metadataRefreshInterval"`

	Metadata    MetadataConfig    `yaml:"metadata"`
	QBittorrent QBittorrentConfig `yaml:"qbittorrent"`
}

type QBittorrentConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Host     string `yaml:"host"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// ValidationErrors holds field-level validation messages.
type ValidationErrors map[string]string

func (e ValidationErrors) Error() string {
	var s string
	for k, v := range e {
		s += k + ": " + v + "; "
	}
	return s
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}

	applyDefaults(cfg)
	applyEnvOverrides(cfg)

	if err := validatePaths(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate performs syntactic / semantic checks only — no filesystem operations.
// Returns nil when valid, or a ValidationErrors with per-field messages.
func (cfg *Config) Validate() error {
	errs := ValidationErrors{}

	if _, err := strconv.Atoi(cfg.Port); cfg.Port != "" && err != nil {
		errs["port"] = "must be a valid integer"
	}
	if cfg.Metadata.EpisodesURL == "" {
		errs["metadataEpisodesUrl"] = "required"
	} else if _, err := url.ParseRequestURI(cfg.Metadata.EpisodesURL); err != nil {
		errs["metadataEpisodesUrl"] = "must be a valid URL"
	}
	if cfg.Metadata.ArcsURL == "" {
		errs["metadataArcsUrl"] = "required"
	} else if _, err := url.ParseRequestURI(cfg.Metadata.ArcsURL); err != nil {
		errs["metadataArcsUrl"] = "must be a valid URL"
	}
	if cfg.MetadataRefreshInterval != "" {
		if _, err := time.ParseDuration(cfg.MetadataRefreshInterval); err != nil {
			errs["metadataRefreshInterval"] = "must be a valid Go duration (e.g. 6h, 30m)"
		}
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// RefreshDuration parses MetadataRefreshInterval, defaulting to 24h.
func (cfg *Config) RefreshDuration() time.Duration {
	if d, err := time.ParseDuration(cfg.MetadataRefreshInterval); err == nil {
		return d
	}
	return 24 * time.Hour
}

func applyDefaults(cfg *Config) {
	if cfg.Port == "" {
		cfg.Port = "8989"
	}
	if cfg.LibraryPath == "" {
		cfg.LibraryPath = "./media"
	}
	if cfg.DownloadPath == "" {
		cfg.DownloadPath = "./downloads"
	}
	if cfg.LibraryJSONPath == "" {
		cfg.LibraryJSONPath = "./data/library.json"
	}
	if cfg.MetadataRefreshInterval == "" {
		cfg.MetadataRefreshInterval = "24h"
	}
}

func applyEnvOverrides(cfg *Config) {
	if port := os.Getenv("OP_PORT"); port != "" {
		cfg.Port = port
	}
	if path := os.Getenv("OP_LIBRARY"); path != "" {
		cfg.LibraryPath = path
	}
	if path := os.Getenv("OP_DOWNLOADS"); path != "" {
		cfg.DownloadPath = path
	}
	if url := os.Getenv("OP_METADATA_EPISODES_URL"); url != "" {
		cfg.Metadata.EpisodesURL = url
	}
	if url := os.Getenv("OP_METADATA_ARCS_URL"); url != "" {
		cfg.Metadata.ArcsURL = url
	}
	if host := os.Getenv("OP_QB_HOST"); host != "" {
		cfg.QBittorrent.Host = host
	}
	if user := os.Getenv("OP_QB_USER"); user != "" {
		cfg.QBittorrent.Username = user
	}
	if pass := os.Getenv("OP_QB_PASS"); pass != "" {
		cfg.QBittorrent.Password = pass
	}
}

// validatePaths checks filesystem accessibility and creates directories if missing.
// Called only at startup — not part of the public Validate() method.
func validatePaths(cfg *Config) error {
	for _, p := range []string{cfg.LibraryPath, cfg.DownloadPath} {
		if _, err := os.Stat(p); err != nil {
			if os.IsNotExist(err) {
				if mkErr := os.MkdirAll(p, 0755); mkErr != nil {
					return fmt.Errorf("path %q could not be created: %w", p, mkErr)
				}
			} else {
				return fmt.Errorf("path %q not accessible: %w", p, err)
			}
		}
	}
	if cfg.Metadata.EpisodesURL == "" {
		return errors.New("metadata.episodesUrl must be set in config.yml")
	}
	if cfg.Metadata.ArcsURL == "" {
		return errors.New("metadata.arcsUrl must be set in config.yml")
	}
	return nil
}

// Save writes cfg back to path as YAML.
func Save(path string, cfg *Config) error {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, b, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

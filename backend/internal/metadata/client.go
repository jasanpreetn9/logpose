package metadata

import (
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Client struct {
	EpisodesURL string
	ArcsURL     string

	Cache *Cache
	http  *http.Client
}

type Cache struct {
	mu            sync.RWMutex
	EpisodesByCRC map[string]Episode
	ArcsByNumber  map[int]Arc
	LastUpdated   time.Time

	lastEpisodesHash [32]byte
	lastArcsHash     [32]byte
	changedCRCs      []string
}

func NewClient(episodesURL, arcsURL string) *Client {
	return &Client{
		EpisodesURL: episodesURL,
		ArcsURL:     arcsURL,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
		Cache: &Cache{
			EpisodesByCRC: map[string]Episode{},
			ArcsByNumber:  map[int]Arc{},
		},
	}
}

func (c *Client) fetchBytes(url string) ([]byte, error) {
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *Client) Refresh() error {
	epBytes, err := c.fetchBytes(c.EpisodesURL)
	if err != nil {
		return err
	}
	arcBytes, err := c.fetchBytes(c.ArcsURL)
	if err != nil {
		return err
	}

	epHash := sha256.Sum256(epBytes)
	arcHash := sha256.Sum256(arcBytes)

	var rawEpisodes map[string]Episode
	if err := json.Unmarshal(epBytes, &rawEpisodes); err != nil {
		return err
	}
	var rawArcs []Arc
	if err := json.Unmarshal(arcBytes, &rawArcs); err != nil {
		return err
	}

	newEpisodesByCRC := map[string]Episode{}
	for key, ep := range rawEpisodes {
		k := strings.ToUpper(key)
		ep.File.CRC32 = strings.ToUpper(ep.File.CRC32)
		newEpisodesByCRC[k] = ep
	}
	newArcsByNumber := map[int]Arc{}
	for _, a := range rawArcs {
		newArcsByNumber[a.ArcNumber] = a
	}

	c.Cache.mu.Lock()
	defer c.Cache.mu.Unlock()

	// Detect which CRC32s changed.
	var changed []string
	if epHash != c.Cache.lastEpisodesHash {
		for crc, newEp := range newEpisodesByCRC {
			old, exists := c.Cache.EpisodesByCRC[crc]
			if !exists || old.File.URL != newEp.File.URL || old.Title != newEp.Title {
				changed = append(changed, crc)
			}
		}
	}

	c.Cache.EpisodesByCRC = newEpisodesByCRC
	c.Cache.ArcsByNumber = newArcsByNumber
	c.Cache.LastUpdated = time.Now()
	c.Cache.lastEpisodesHash = epHash
	c.Cache.lastArcsHash = arcHash
	if len(changed) > 0 {
		c.Cache.changedCRCs = append(c.Cache.changedCRCs, changed...)
	}

	return nil
}

// StaleEpisodes returns CRC32s whose metadata changed since the last refresh,
// then clears the internal list.
func (c *Client) StaleEpisodes() []string {
	c.Cache.mu.Lock()
	defer c.Cache.mu.Unlock()
	out := make([]string, len(c.Cache.changedCRCs))
	copy(out, c.Cache.changedCRCs)
	c.Cache.changedCRCs = nil
	return out
}

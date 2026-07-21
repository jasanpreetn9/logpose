package qbittorrent

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type TorrentInfo struct {
	Name        string  `json:"name"`
	SavePath    string  `json:"save_path"`
	ContentPath string  `json:"content_path"`
	Hash        string  `json:"hash"`
	Progress    float64 `json:"progress"`
	Size        int64   `json:"size"`
	DLSpeed     int64   `json:"dlspeed"`
	ETA         int64   `json:"eta"`
	State       string  `json:"state"`
}

// Completed reports whether the torrent has finished downloading.
func (t TorrentInfo) Completed() bool {
	return t.Progress >= 1
}

// GetTorrents returns all torrents in the Logpose category, complete or not.
// Torrents added by other applications are never returned.
func (c *Client) GetTorrents() ([]TorrentInfo, error) {
	if c.Cookie == "" {
		if err := c.Login(); err != nil {
			return nil, err
		}
	}

	result, err := c.getTorrentsRequest()
	if err == errSessionExpired {
		c.Cookie = ""
		if loginErr := c.Login(); loginErr != nil {
			return nil, loginErr
		}
		return c.getTorrentsRequest()
	}
	return result, err
}

func (c *Client) getTorrentsRequest() ([]TorrentInfo, error) {
	resp, err := c.makeRequest("GET", "/api/v2/torrents/info?category="+url.QueryEscape(Category), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 || resp.StatusCode == 401 {
		return nil, errSessionExpired
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GetTorrents: status %d", resp.StatusCode)
	}

	var torrents []TorrentInfo
	if err := json.NewDecoder(resp.Body).Decode(&torrents); err != nil {
		return nil, err
	}
	return torrents, nil
}

// DeleteTorrent removes a torrent from qBittorrent (does not delete files).
func (c *Client) DeleteTorrent(hash string) error {
	if c.Cookie == "" {
		if err := c.Login(); err != nil {
			return err
		}
	}

	form := url.Values{}
	form.Set("hashes", hash)
	form.Set("deleteFiles", "false")

	resp, err := c.makeRequest("POST", "/api/v2/torrents/delete", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 || resp.StatusCode == 401 {
		c.Cookie = ""
		if loginErr := c.Login(); loginErr != nil {
			return loginErr
		}
		resp2, err := c.makeRequest("POST", "/api/v2/torrents/delete", strings.NewReader(form.Encode()))
		if err != nil {
			return err
		}
		resp2.Body.Close()
	}
	return nil
}

package qbittorrent

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type TorrentInfo struct {
	Name        string `json:"name"`
	SavePath    string `json:"save_path"`
	ContentPath string `json:"content_path"`
	Hash        string `json:"hash"`
}

// GetCompleted returns all torrents with filter=completed.
func (c *Client) GetCompleted() ([]TorrentInfo, error) {
	if c.Cookie == "" {
		if err := c.Login(); err != nil {
			return nil, err
		}
	}

	result, err := c.getCompletedRequest()
	if err == errSessionExpired {
		c.Cookie = ""
		if loginErr := c.Login(); loginErr != nil {
			return nil, loginErr
		}
		return c.getCompletedRequest()
	}
	return result, err
}

func (c *Client) getCompletedRequest() ([]TorrentInfo, error) {
	resp, err := c.makeRequest("GET", "/api/v2/torrents/info?filter=completed", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 403 || resp.StatusCode == 401 {
		return nil, errSessionExpired
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GetCompleted: status %d", resp.StatusCode)
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

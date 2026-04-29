package qbittorrent

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// nyaaViewRe matches Nyaa view-page URLs so they can be rewritten to direct
// torrent download URLs that qBittorrent can actually fetch.
var nyaaViewRe = regexp.MustCompile(`^https?://nyaa\.si/view/(\d+)`)

func (c *Client) AddTorrent(downloadURL string) error {
	if c.Cookie == "" {
		if err := c.Login(); err != nil {
			return err
		}
	}

	if err := c.addTorrentRequest(downloadURL); err != nil {
		// Session may have expired — clear cookie and retry once.
		c.Cookie = ""
		if loginErr := c.Login(); loginErr != nil {
			return loginErr
		}
		return c.addTorrentRequest(downloadURL)
	}
	return nil
}

// normalizeTorrentURL converts a Nyaa view page URL to its direct .torrent URL.
// All other URLs are returned unchanged.
func normalizeTorrentURL(rawURL string) string {
	if m := nyaaViewRe.FindStringSubmatch(rawURL); m != nil {
		return "https://nyaa.si/download/" + m[1] + ".torrent"
	}
	return rawURL
}

func (c *Client) addTorrentRequest(downloadURL string) error {
	downloadURL = normalizeTorrentURL(downloadURL)
	form := url.Values{}
	form.Set("urls", downloadURL)

	req, err := http.NewRequest(
		"POST",
		c.Host+"/api/v2/torrents/add",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", c.Cookie)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		return errSessionExpired
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qbittorrent add failed: %d", resp.StatusCode)
	}
	// qBittorrent returns "Ok." on success and "Fails." on rejection (e.g. bad URL,
	// duplicate torrent) — both with HTTP 200.
	bodyStr := strings.TrimSpace(string(body))
	if bodyStr != "Ok." {
		return fmt.Errorf("qbittorrent rejected torrent: %q", bodyStr)
	}
	return nil
}

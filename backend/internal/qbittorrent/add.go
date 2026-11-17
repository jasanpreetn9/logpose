package qbittorrent

import (
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
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

// errAlreadyAdded means qBittorrent rejected the add because it already has
// the torrent. That is success from the caller's point of view: a batch
// release (one torrent covering a whole arc) is the download link for every
// episode in it, so queueing a second episode from the same batch re-adds
// the same magnet.
var errAlreadyAdded = errors.New("torrent already in qBittorrent")

// Category is assigned to every torrent Logpose adds, so the completion
// poller only ever touches torrents that belong to Logpose.
const Category = "logpose"

func (c *Client) AddTorrent(downloadURL string) error {
	if c.Cookie == "" {
		if err := c.Login(); err != nil {
			return err
		}
	}

	err := c.addTorrentRequest(downloadURL)
	if err == nil || errors.Is(err, errAlreadyAdded) {
		return nil
	}

	// Session may have expired — clear cookie and retry once.
	c.Cookie = ""
	if loginErr := c.Login(); loginErr != nil {
		return loginErr
	}
	err = c.addTorrentRequest(downloadURL)
	if errors.Is(err, errAlreadyAdded) {
		return nil
	}
	return err
}

// normalizeTorrentURL converts a Nyaa view page URL to its direct .torrent URL.
// All other URLs are returned unchanged.
func normalizeTorrentURL(rawURL string) string {
	if m := nyaaViewRe.FindStringSubmatch(rawURL); m != nil {
		return "https://nyaa.si/download/" + m[1] + ".torrent"
	}
	return rawURL
}

// ensureCategory creates the Logpose category if it doesn't exist yet.
// qBittorrent returns 409 when it already exists — both count as success.
func (c *Client) ensureCategory() {
	form := url.Values{}
	form.Set("category", Category)
	form.Set("savePath", "")
	resp, err := c.makeRequest("POST", "/api/v2/torrents/createCategory", strings.NewReader(form.Encode()))
	if err == nil {
		resp.Body.Close()
	}
}

func (c *Client) addTorrentRequest(downloadURL string) error {
	c.ensureCategory()
	downloadURL = normalizeTorrentURL(downloadURL)
	form := url.Values{}
	form.Set("urls", downloadURL)
	form.Set("category", Category)

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
		if hash := magnetInfoHash(downloadURL); hash != "" {
			if exists, err := c.hasTorrent(hash); err == nil && exists {
				return errAlreadyAdded
			}
		}
		return fmt.Errorf("qbittorrent rejected torrent: %q", bodyStr)
	}
	return nil
}

// hasTorrent reports whether qBittorrent already has a torrent with this
// info hash, in any category.
func (c *Client) hasTorrent(hash string) (bool, error) {
	resp, err := c.makeRequest("GET", "/api/v2/torrents/info?hashes="+url.QueryEscape(hash), nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("torrents/info: status %d", resp.StatusCode)
	}
	var torrents []TorrentInfo
	if err := json.NewDecoder(resp.Body).Decode(&torrents); err != nil {
		return false, err
	}
	return len(torrents) > 0, nil
}

// magnetInfoHash returns the lowercase hex BitTorrent v1 info hash of a
// magnet URI's "xt=urn:btih:" parameter (hex or base32 form), or "" if the
// URL isn't a magnet or carries no v1 hash.
func magnetInfoHash(link string) string {
	if !strings.HasPrefix(link, "magnet:?") {
		return ""
	}
	values, err := url.ParseQuery(strings.TrimPrefix(link, "magnet:?"))
	if err != nil {
		return ""
	}
	for _, xt := range values["xt"] {
		const prefix = "urn:btih:"
		if !strings.HasPrefix(strings.ToLower(xt), prefix) {
			continue
		}
		h := xt[len(prefix):]
		switch len(h) {
		case 40:
			if _, err := hex.DecodeString(h); err == nil {
				return strings.ToLower(h)
			}
		case 32:
			if b, err := base32.StdEncoding.DecodeString(strings.ToUpper(h)); err == nil {
				return hex.EncodeToString(b)
			}
		}
	}
	return ""
}

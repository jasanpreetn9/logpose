package qbittorrent

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var errSessionExpired = errors.New("qbittorrent session expired")

type Client struct {
	Host     string
	Username string
	Password string
	Cookie   string

	http *http.Client
}

func NewClient(host, username, password string) *Client {
	return &Client{
		Host:     strings.TrimRight(host, "/"),
		Username: username,
		Password: password,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Login and store the SID cookie
func (c *Client) Login() error {
	loginURL := c.Host + "/api/v2/auth/login"

	data := url.Values{}
	data.Set("username", c.Username)
	data.Set("password", c.Password)

	req, err := http.NewRequest("POST", loginURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("qbit login failed: %d", resp.StatusCode)
	}

	// Extract SID cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "SID" {
			c.Cookie = "SID=" + cookie.Value
			return nil
		}
	}

	return errors.New("qbittorrent did not provide SID cookie")
}

// makeRequest with cookie
func (c *Client) makeRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.Host+endpoint, body)
	if err != nil {
		return nil, err
	}

	if c.Cookie != "" {
		req.Header.Set("Cookie", c.Cookie)
	}

	return c.http.Do(req)
}

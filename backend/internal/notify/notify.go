// Package notify sends best-effort external notifications (Discord, Jellyfin)
// when the activity log records an import or a failure. Every method is a
// no-op when its corresponding config URL is empty, so notifications are
// opt-in per-integration.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"onepace-library/internal/config"
)

// jellyfinDebounce is how long to wait after the last import before
// triggering a Jellyfin library refresh, so a batch of auto-grabbed imports
// collapses into a single refresh call.
const jellyfinDebounce = 20 * time.Second

const (
	colorGreen = 3066993
	colorRed   = 15158332
)

type Notifier struct {
	cfg  *config.Config // read fields fresh on every call, never cached
	http *http.Client

	jellyfinTrigger chan struct{}
}

// New starts the Jellyfin debounce worker, which runs until ctx is cancelled.
func New(ctx context.Context, cfg *config.Config) *Notifier {
	n := &Notifier{
		cfg:             cfg,
		http:            &http.Client{Timeout: 10 * time.Second},
		jellyfinTrigger: make(chan struct{}, 1),
	}
	go n.debounceJellyfin(ctx)
	return n
}

// NotifyImport reports a successful import to Discord and schedules a
// debounced Jellyfin library refresh. message is the activity event's
// message (e.g. "Imported: Enter: Nami"), already human-readable.
func (n *Notifier) NotifyImport(message, path string) {
	n.postDiscord(message, path, colorGreen)
	if n.cfg.Notifications.JellyfinURL != "" {
		select {
		case n.jellyfinTrigger <- struct{}{}:
		default:
		}
	}
}

// NotifyFailure reports a failed download/import to Discord only. message is
// the activity event's message (e.g. "Download failed: X").
func (n *Notifier) NotifyFailure(message, reason string) {
	n.postDiscord(message, reason, colorRed)
}

func (n *Notifier) postDiscord(title, detail string, color int) {
	url := n.cfg.Notifications.DiscordWebhookURL
	if url == "" {
		return
	}

	body, err := json.Marshal(map[string]any{
		"embeds": []map[string]any{
			{
				"title":       title,
				"description": detail,
				"color":       color,
				"timestamp":   time.Now().UTC().Format(time.RFC3339),
			},
		},
	})
	if err != nil {
		log.Printf("notify: marshal discord payload: %v", err)
		return
	}

	resp, err := n.http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("notify: discord webhook: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("notify: discord webhook: status %d", resp.StatusCode)
	}
}

func (n *Notifier) debounceJellyfin(ctx context.Context) {
	var timer *time.Timer
	var fire <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-n.jellyfinTrigger:
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(jellyfinDebounce)
			fire = timer.C
		case <-fire:
			fire = nil
			n.refreshJellyfin()
		}
	}
}

func (n *Notifier) refreshJellyfin() {
	url := n.cfg.Notifications.JellyfinURL
	if url == "" {
		return
	}

	req, err := http.NewRequest(http.MethodPost, url+"/Library/Refresh", nil)
	if err != nil {
		log.Printf("notify: build jellyfin request: %v", err)
		return
	}
	req.Header.Set("X-Emby-Token", n.cfg.Notifications.JellyfinAPIKey)

	resp, err := n.http.Do(req)
	if err != nil {
		log.Printf("notify: jellyfin refresh: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("notify: jellyfin refresh: status %d", resp.StatusCode)
		return
	}
	log.Println("notify: triggered Jellyfin library refresh")
}

// Package downloads tracks episode versions that are currently queued or
// downloading in qBittorrent, keyed by CRC32.
package downloads

import (
	"strings"
	"sync"
	"time"
)

// manualTTL covers the gap between queueing a torrent via the API and the
// poller first observing it in qBittorrent.
const manualTTL = 2 * time.Minute

type Tracker struct {
	mu     sync.Mutex
	manual map[string]time.Time
	active map[string]struct{}
}

func NewTracker() *Tracker {
	return &Tracker{
		manual: map[string]time.Time{},
		active: map[string]struct{}{},
	}
}

// MarkQueued records a CRC as just queued via the API.
func (t *Tracker) MarkQueued(crc string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.manual[strings.ToUpper(crc)] = time.Now()
}

// SetActive replaces the set of CRCs observed as incomplete torrents in
// qBittorrent. Manual marks the poller has confirmed (or that expired) are
// dropped.
func (t *Tracker) SetActive(crcs []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.active = make(map[string]struct{}, len(crcs))
	for _, c := range crcs {
		t.active[strings.ToUpper(c)] = struct{}{}
	}
	for c, at := range t.manual {
		if _, confirmed := t.active[c]; confirmed || time.Since(at) > manualTTL {
			delete(t.manual, c)
		}
	}
}

// IsActive reports whether the CRC is queued or downloading.
func (t *Tracker) IsActive(crc string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	crc = strings.ToUpper(crc)
	if _, ok := t.active[crc]; ok {
		return true
	}
	at, ok := t.manual[crc]
	return ok && time.Since(at) <= manualTTL
}

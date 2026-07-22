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

type importState struct {
	bytesTotal int64
	bytesDone  int64
}

type Tracker struct {
	mu        sync.Mutex
	manual    map[string]time.Time
	active    map[string]struct{}
	importing map[string]*importState
}

func NewTracker() *Tracker {
	return &Tracker{
		manual:    map[string]time.Time{},
		active:    map[string]struct{}{},
		importing: map[string]*importState{},
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

// SetImporting marks a CRC as being moved from the downloads folder into the
// library, with bytesTotal for progress reporting (0 if unknown).
func (t *Tracker) SetImporting(crc string, bytesTotal int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.importing[strings.ToUpper(crc)] = &importState{bytesTotal: bytesTotal}
}

// UpdateImportProgress records bytes copied so far for a CRC already marked
// importing. No-op if the CRC isn't currently importing.
func (t *Tracker) UpdateImportProgress(crc string, bytesDone int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if s, ok := t.importing[strings.ToUpper(crc)]; ok {
		s.bytesDone = bytesDone
	}
}

// ClearImporting removes the importing mark for a CRC, e.g. once the move
// finishes (successfully or not).
func (t *Tracker) ClearImporting(crc string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.importing, strings.ToUpper(crc))
}

// IsImporting reports whether the CRC is currently being moved into the library.
func (t *Tracker) IsImporting(crc string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	_, ok := t.importing[strings.ToUpper(crc)]
	return ok
}

// ImportProgress returns the fraction (0..1) of the move completed so far.
// Returns 0 if the CRC isn't importing or the total size is unknown.
func (t *Tracker) ImportProgress(crc string) float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	s, ok := t.importing[strings.ToUpper(crc)]
	if !ok || s.bytesTotal <= 0 {
		return 0
	}
	frac := float64(s.bytesDone) / float64(s.bytesTotal)
	if frac > 1 {
		return 1
	}
	return frac
}

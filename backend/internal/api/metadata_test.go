package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
)

const syncTestEpisodesJSON = `{
	"aaaa1111": {"arc": 1, "episode": 1, "title": "Romance Dawn 01", "file": {"version": "normal", "crc32": "aaaa1111", "url": "https://example.com/a"}},
	"bbbb2222": {"arc": 1, "episode": 2, "title": "Romance Dawn 02", "file": {"version": "normal", "crc32": "bbbb2222", "url": "https://example.com/b"}}
}`

const syncTestArcsJSON = `[{"arc": 1, "title": "Romance Dawn"}]`

func newTestMetadataClient(t *testing.T) *metadata.Client {
	t.Helper()
	epServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(syncTestEpisodesJSON))
	}))
	t.Cleanup(epServer.Close)
	arcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(syncTestArcsJSON))
	}))
	t.Cleanup(arcServer.Close)

	c := metadata.NewClient(epServer.URL, arcServer.URL)
	if err := c.Refresh(); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	return c
}

// Regression test for the monitor/episode desync bug: an arc monitored
// before episode 2 existed in its library entry must have episode 2
// backfilled as monitored once metadata catches up, otherwise it silently
// never surfaces on Wanted or gets auto-grabbed.
func TestSyncMonitoredEpisodes_BackfillsMissingEpisodeOnMonitoredArc(t *testing.T) {
	store := newTestStore(t)
	meta := newTestMetadataClient(t)

	if err := store.Write(func(lib *library.Library) error {
		arc := lib.GetOrCreateArc(1, "Romance Dawn")
		arc.Monitored = true
		arc.Episodes["1"] = library.Episode{
			EpisodeNumber: 1,
			Monitored:     true,
			Versions:      map[string]library.EpisodeVersion{},
		}
		// Episode 2 deliberately absent, simulating it not existing in
		// metadata yet when the arc was first monitored.
		return nil
	}); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	backfilled := SyncMonitoredEpisodes(meta, store)
	if backfilled != 1 {
		t.Fatalf("backfilled = %d, want 1", backfilled)
	}

	store.Read(func(lib *library.Library) {
		ep, ok := lib.Arcs[1].Episodes["2"]
		if !ok {
			t.Fatal("expected episode 2 to be created")
		}
		if !ep.Monitored {
			t.Error("expected backfilled episode to be monitored")
		}
	})
}

// An unmonitored arc must never have episodes backfilled — only arcs the
// user explicitly monitored should gain new episode entries automatically.
func TestSyncMonitoredEpisodes_SkipsUnmonitoredArc(t *testing.T) {
	store := newTestStore(t)
	meta := newTestMetadataClient(t)

	if err := store.Write(func(lib *library.Library) error {
		arc := lib.GetOrCreateArc(1, "Romance Dawn")
		arc.Monitored = false
		return nil
	}); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	if backfilled := SyncMonitoredEpisodes(meta, store); backfilled != 0 {
		t.Errorf("backfilled = %d, want 0 for unmonitored arc", backfilled)
	}

	store.Read(func(lib *library.Library) {
		if len(lib.Arcs[1].Episodes) != 0 {
			t.Errorf("expected no episodes created, got %v", lib.Arcs[1].Episodes)
		}
	})
}

// Episodes that already exist must be left untouched (in particular, an
// already-imported or explicitly-unmonitored episode must not be reset).
func TestSyncMonitoredEpisodes_DoesNotTouchExistingEpisodes(t *testing.T) {
	store := newTestStore(t)
	meta := newTestMetadataClient(t)

	if err := store.Write(func(lib *library.Library) error {
		arc := lib.GetOrCreateArc(1, "Romance Dawn")
		arc.Monitored = true
		arc.Episodes["1"] = library.Episode{
			EpisodeNumber: 1,
			Monitored:     false, // user explicitly unmonitored this one episode
			Versions:      map[string]library.EpisodeVersion{},
		}
		return nil
	}); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	SyncMonitoredEpisodes(meta, store)

	store.Read(func(lib *library.Library) {
		if lib.Arcs[1].Episodes["1"].Monitored {
			t.Error("expected existing episode's monitored flag to be left untouched")
		}
	})
}

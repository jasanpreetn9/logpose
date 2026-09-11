package metadata

import (
	"testing"
	"time"
)

func newPopulatedClient() *Client {
	c := NewClient("", "")
	c.Cache.EpisodesByCRC = map[string]Episode{
		"AAAA1111": {Arc: 1, Episode: 1, Title: "Romance Dawn 01"},
		"BBBB2222": {Arc: 1, Episode: 2, Title: "Romance Dawn 02"},
		"CCCC3333": {Arc: 1, Episode: 2, Title: "Romance Dawn 02 Extended"}, // same episode, different version/CRC
		"DDDD4444": {Arc: 2, Episode: 1, Title: "Orange Town 01"},
	}
	c.Cache.ArcsByNumber = map[int]Arc{
		1: {ArcNumber: 1, Title: "Romance Dawn"},
	}
	return c
}

func TestGetEpisodeByCRC32(t *testing.T) {
	c := newPopulatedClient()

	ep, err := c.GetEpisodeByCRC32("AAAA1111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ep.Title != "Romance Dawn 01" {
		t.Errorf("Title = %q, want %q", ep.Title, "Romance Dawn 01")
	}

	if _, err := c.GetEpisodeByCRC32("DEADBEEF"); err == nil {
		t.Error("expected error for unknown CRC, got nil")
	}
}

func TestGetArcTitle(t *testing.T) {
	c := newPopulatedClient()

	if got := c.GetArcTitle(1); got != "Romance Dawn" {
		t.Errorf("GetArcTitle(1) = %q, want %q", got, "Romance Dawn")
	}
	// Falls back to a generic label rather than erroring, since callers
	// (the scanner) need something to build a folder/NFO title from even
	// when arc metadata hasn't loaded yet.
	if got := c.GetArcTitle(99); got != "Arc 99" {
		t.Errorf("GetArcTitle(99) = %q, want %q", got, "Arc 99")
	}
}

func TestGetArcByNumber(t *testing.T) {
	c := newPopulatedClient()

	arc, err := c.GetArcByNumber(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if arc.Title != "Romance Dawn" {
		t.Errorf("Title = %q, want %q", arc.Title, "Romance Dawn")
	}

	if _, err := c.GetArcByNumber(99); err == nil {
		t.Error("expected error for unknown arc, got nil")
	}
}

func TestEpisodesByArc_FiltersAndDedupesByEpisodeNumber(t *testing.T) {
	c := newPopulatedClient()

	got := c.EpisodesByArc(1)
	if len(got) != 2 {
		t.Fatalf("expected 2 distinct episodes for arc 1 (deduped), got %d: %+v", len(got), got)
	}
	seen := map[int]bool{}
	for _, ep := range got {
		if ep.Arc != 1 {
			t.Errorf("got episode from arc %d, want only arc 1", ep.Arc)
		}
		seen[ep.Episode] = true
	}
	if !seen[1] || !seen[2] {
		t.Errorf("expected episodes 1 and 2, got %v", got)
	}
}

func TestEpisodesByArc_NoMatches(t *testing.T) {
	c := newPopulatedClient()
	if got := c.EpisodesByArc(99); len(got) != 0 {
		t.Errorf("expected no episodes for unknown arc, got %+v", got)
	}
}

func TestCounts(t *testing.T) {
	c := newPopulatedClient()
	episodes, arcs := c.Counts()
	if episodes != 4 {
		t.Errorf("episodes = %d, want 4", episodes)
	}
	if arcs != 1 {
		t.Errorf("arcs = %d, want 1", arcs)
	}
}

func TestLastUpdated(t *testing.T) {
	c := NewClient("", "")
	if !c.LastUpdated().IsZero() {
		t.Error("expected zero time before any refresh")
	}
	now := time.Now()
	c.Cache.LastUpdated = now
	if got := c.LastUpdated(); !got.Equal(now) {
		t.Errorf("LastUpdated() = %v, want %v", got, now)
	}
}

// Episodes() must return an independent copy — callers iterating it (e.g.
// the API layer merging with library state) must not be able to mutate the
// client's cache out from under concurrent readers.
func TestEpisodes_ReturnsIndependentSnapshot(t *testing.T) {
	c := newPopulatedClient()

	snapshot := c.Episodes()
	if len(snapshot) != 4 {
		t.Fatalf("expected 4 episodes in snapshot, got %d", len(snapshot))
	}

	delete(snapshot, "AAAA1111")
	snapshot["ZZZZ9999"] = Episode{Title: "injected"}

	if _, ok := c.Cache.EpisodesByCRC["AAAA1111"]; !ok {
		t.Error("mutating the snapshot deleted an entry from the real cache")
	}
	if _, ok := c.Cache.EpisodesByCRC["ZZZZ9999"]; ok {
		t.Error("mutating the snapshot added an entry to the real cache")
	}
}

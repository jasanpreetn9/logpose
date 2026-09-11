package metadata

import (
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
)

// servedClient wires a Client to two httptest.Servers whose response bodies
// can be swapped between calls by writing to *episodesBody / *arcsBody.
func servedClient(t *testing.T, episodesBody, arcsBody *string) *Client {
	t.Helper()
	epServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(*episodesBody))
	}))
	t.Cleanup(epServer.Close)
	arcServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(*arcsBody))
	}))
	t.Cleanup(arcServer.Close)
	return NewClient(epServer.URL, arcServer.URL)
}

func sortedStrings(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}

const sampleEpisodesJSON = `{
	"aaaa1111": {"arc": 1, "episode": 1, "title": "Romance Dawn 01", "file": {"version": "normal", "crc32": "aaaa1111", "url": "https://example.com/a"}},
	"bbbb2222": {"arc": 1, "episode": 2, "title": "Romance Dawn 02", "file": {"version": "normal", "crc32": "bbbb2222", "url": "https://example.com/b"}}
}`

const sampleArcsJSON = `[{"arc": 1, "title": "Romance Dawn"}]`

func TestRefresh_PopulatesCache(t *testing.T) {
	ep, arc := sampleEpisodesJSON, sampleArcsJSON
	c := servedClient(t, &ep, &arc)

	if err := c.Refresh(); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	episodes, arcs := c.Counts()
	if episodes != 2 || arcs != 1 {
		t.Fatalf("Counts() = %d, %d; want 2, 1", episodes, arcs)
	}
	if c.LastUpdated().IsZero() {
		t.Error("expected LastUpdated to be set after Refresh")
	}
}

// CRC32 keys and File.CRC32 values must be uppercased on load, since that's
// the canonical form used everywhere else (filenames, library storage).
func TestRefresh_UppercasesCRC32(t *testing.T) {
	ep, arc := sampleEpisodesJSON, sampleArcsJSON
	c := servedClient(t, &ep, &arc)

	if err := c.Refresh(); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	got, err := c.GetEpisodeByCRC32("AAAA1111")
	if err != nil {
		t.Fatalf("expected uppercase-keyed lookup to succeed: %v", err)
	}
	if got.File.CRC32 != "AAAA1111" {
		t.Errorf("File.CRC32 = %q, want uppercased %q", got.File.CRC32, "AAAA1111")
	}
}

// On the very first refresh there's nothing to diff against, so every
// episode is reported stale — this drives the initial NFO generation pass.
func TestRefresh_FirstRefreshMarksAllEpisodesStale(t *testing.T) {
	ep, arc := sampleEpisodesJSON, sampleArcsJSON
	c := servedClient(t, &ep, &arc)

	if err := c.Refresh(); err != nil {
		t.Fatalf("Refresh: %v", err)
	}

	stale := sortedStrings(c.StaleEpisodes())
	want := []string{"AAAA1111", "BBBB2222"}
	if len(stale) != len(want) {
		t.Fatalf("StaleEpisodes() = %v, want %v", stale, want)
	}
	for i := range want {
		if stale[i] != want[i] {
			t.Errorf("StaleEpisodes()[%d] = %q, want %q", i, stale[i], want[i])
		}
	}
}

func TestRefresh_IdenticalContentProducesNoNewStaleEpisodes(t *testing.T) {
	ep, arc := sampleEpisodesJSON, sampleArcsJSON
	c := servedClient(t, &ep, &arc)

	if err := c.Refresh(); err != nil {
		t.Fatalf("Refresh 1: %v", err)
	}
	c.StaleEpisodes() // drain the initial "everything is new" batch

	if err := c.Refresh(); err != nil {
		t.Fatalf("Refresh 2: %v", err)
	}
	if stale := c.StaleEpisodes(); len(stale) != 0 {
		t.Errorf("expected no stale episodes for an unchanged refresh, got %v", stale)
	}
}

func TestRefresh_OnlyReportsTheEpisodeThatActuallyChanged(t *testing.T) {
	ep, arc := sampleEpisodesJSON, sampleArcsJSON
	c := servedClient(t, &ep, &arc)

	if err := c.Refresh(); err != nil {
		t.Fatalf("Refresh 1: %v", err)
	}
	c.StaleEpisodes()

	ep = `{
		"aaaa1111": {"arc": 1, "episode": 1, "title": "Romance Dawn 01 (Retimed)", "file": {"version": "normal", "crc32": "aaaa1111", "url": "https://example.com/a"}},
		"bbbb2222": {"arc": 1, "episode": 2, "title": "Romance Dawn 02", "file": {"version": "normal", "crc32": "bbbb2222", "url": "https://example.com/b"}}
	}`
	if err := c.Refresh(); err != nil {
		t.Fatalf("Refresh 2: %v", err)
	}

	stale := c.StaleEpisodes()
	if len(stale) != 1 || stale[0] != "AAAA1111" {
		t.Errorf("StaleEpisodes() = %v, want exactly [AAAA1111]", stale)
	}
}

// StaleEpisodes must drain its internal list — calling it twice in a row
// without an intervening Refresh should only return the batch once.
func TestStaleEpisodes_DrainsOnce(t *testing.T) {
	ep, arc := sampleEpisodesJSON, sampleArcsJSON
	c := servedClient(t, &ep, &arc)

	if err := c.Refresh(); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	first := c.StaleEpisodes()
	if len(first) == 0 {
		t.Fatal("expected the first drain to return the stale batch")
	}
	second := c.StaleEpisodes()
	if len(second) != 0 {
		t.Errorf("expected the second drain to be empty, got %v", second)
	}
}

func TestRefresh_ServerError(t *testing.T) {
	epServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer epServer.Close()
	c := NewClient(epServer.URL, "")

	if err := c.Refresh(); err == nil {
		t.Error("expected an error when the episodes endpoint 500s, got nil")
	}
	if episodes, _ := c.Counts(); episodes != 0 {
		t.Errorf("expected cache to remain empty after a failed refresh, got %d episodes", episodes)
	}
}

func TestRefresh_MalformedJSON(t *testing.T) {
	ep, arc := "not json", sampleArcsJSON
	c := servedClient(t, &ep, &arc)

	if err := c.Refresh(); err == nil {
		t.Error("expected an error for malformed episodes JSON, got nil")
	}
}

// A failed refresh must leave any previously-loaded cache untouched, so a
// transient network blip doesn't wipe out working metadata.
func TestRefresh_FailurePreservesPreviousCache(t *testing.T) {
	ep, arc := sampleEpisodesJSON, sampleArcsJSON
	c := servedClient(t, &ep, &arc)
	if err := c.Refresh(); err != nil {
		t.Fatalf("Refresh 1: %v", err)
	}

	ep = "not json"
	if err := c.Refresh(); err == nil {
		t.Fatal("expected second refresh to fail")
	}

	episodes, _ := c.Counts()
	if episodes != 2 {
		t.Errorf("expected the previous cache (2 episodes) to survive a failed refresh, got %d", episodes)
	}
}

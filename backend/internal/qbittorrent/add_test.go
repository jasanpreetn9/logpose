package qbittorrent

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const batchMagnet = "magnet:?xt=urn:btih:32EE8ADA81131B41334F399B450D0F6516A38ED6&dn=%5BOne+Pace%5D%5B96-100%5D+Loguetown+%5B1080p%5D"

// fakeQbit is a minimal qBittorrent Web API: login always succeeds; add
// returns "Ok." the first time a URL is seen and "Fails." after that, the
// way qBittorrent rejects a torrent it already has.
func fakeQbit(t *testing.T, knownHashes map[string]bool) (*httptest.Server, *int) {
	t.Helper()
	adds := 0
	added := map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			http.SetCookie(w, &http.Cookie{Name: "SID", Value: "x"})
		case "/api/v2/torrents/createCategory":
		case "/api/v2/torrents/add":
			adds++
			_ = r.ParseForm()
			u := r.Form.Get("urls")
			if added[u] {
				w.Write([]byte("Fails."))
				return
			}
			added[u] = true
			if h := magnetInfoHash(u); h != "" {
				knownHashes[h] = true
			}
			w.Write([]byte("Ok."))
		case "/api/v2/torrents/info":
			if knownHashes[r.URL.Query().Get("hashes")] {
				w.Write([]byte(`[{"name":"x","hash":"` + r.URL.Query().Get("hashes") + `"}]`))
				return
			}
			w.Write([]byte(`[]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, &adds
}

// Queueing a second episode from the same batch torrent re-adds the same
// magnet. qBittorrent answers "Fails."; that must count as success.
func TestAddTorrent_DuplicateBatchIsSuccess(t *testing.T) {
	srv, _ := fakeQbit(t, map[string]bool{})
	c := NewClient(srv.URL, "u", "p")

	if err := c.AddTorrent(batchMagnet); err != nil {
		t.Fatalf("first add: %v", err)
	}
	if err := c.AddTorrent(batchMagnet); err != nil {
		t.Fatalf("second add of the same batch should succeed, got %v", err)
	}
}

// A rejection for a torrent qBittorrent does not have (bad link, etc.)
// is still an error.
func TestAddTorrent_RealRejectionStillFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/auth/login":
			http.SetCookie(w, &http.Cookie{Name: "SID", Value: "x"})
		case "/api/v2/torrents/add":
			w.Write([]byte("Fails."))
		case "/api/v2/torrents/info":
			w.Write([]byte(`[]`))
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "u", "p")

	err := c.AddTorrent(batchMagnet)
	if err == nil || !strings.Contains(err.Error(), "rejected") {
		t.Fatalf("want a rejection error, got %v", err)
	}
}

func TestMagnetInfoHash(t *testing.T) {
	cases := map[string]string{
		batchMagnet: "32ee8ada81131b41334f399b450d0f6516a38ed6",
		// Same hash in base32 form.
		"magnet:?xt=urn:btih:GLXIVWUBCMNUCM2PHGNUKDIPMULKHDWW": "32ee8ada81131b41334f399b450d0f6516a38ed6",
		"https://nyaa.si/download/1.torrent":                   "",
		"magnet:?dn=no-hash":                                   "",
		"magnet:?xt=urn:btih:nothex":                           "",
	}
	for in, want := range cases {
		if got := magnetInfoHash(in); got != want {
			t.Errorf("magnetInfoHash(%q) = %q, want %q", in, got, want)
		}
	}
}

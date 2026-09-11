package poller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"onepace-library/internal/activity"
	"onepace-library/internal/db"
	"onepace-library/internal/downloads"
	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
	"onepace-library/internal/qbittorrent"
	"onepace-library/internal/scanner"
	"onepace-library/internal/sse"
)

func newTestStore(t *testing.T) *library.Store {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return library.NewStore("", d)
}

func newTestActivityStore(t *testing.T) *activity.Store {
	t.Helper()
	d, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return activity.NewStore(d, sse.NewHub())
}

func newTestMetaClient(episodes map[string]metadata.Episode) *metadata.Client {
	c := metadata.NewClient("", "")
	c.Cache.EpisodesByCRC = episodes
	c.Cache.ArcsByNumber = map[int]metadata.Arc{1: {ArcNumber: 1, Title: "Romance Dawn"}}
	return c
}

// mockQbit spins up a qBittorrent-like server backed by a mutable torrent
// list, so tests can drive poll()/importTorrent() without a real qBittorrent
// instance. The client's Cookie is pre-set so Login is never invoked.
type mockQbit struct {
	mu             sync.Mutex
	torrents       []qbittorrent.TorrentInfo
	deletedHashes  []string
	infoStatusCode int
}

func newMockQbit(t *testing.T) (*mockQbit, *qbittorrent.Client) {
	t.Helper()
	m := &mockQbit{infoStatusCode: http.StatusOK}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/torrents/info", func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.infoStatusCode != http.StatusOK {
			w.WriteHeader(m.infoStatusCode)
			return
		}
		json.NewEncoder(w).Encode(m.torrents)
	})
	mux.HandleFunc("/api/v2/torrents/delete", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		m.mu.Lock()
		m.deletedHashes = append(m.deletedHashes, r.FormValue("hashes"))
		m.mu.Unlock()
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	qb := qbittorrent.NewClient(server.URL, "user", "pass")
	qb.Cookie = "SID=fake" // skip Login entirely

	return m, qb
}

func (m *mockQbit) setTorrents(torrents []qbittorrent.TorrentInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.torrents = torrents
}

func (m *mockQbit) wasDeleted(hash string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, h := range m.deletedHashes {
		if h == hash {
			return true
		}
	}
	return false
}

// --- importFile -------------------------------------------------------

func TestImportFile_Success(t *testing.T) {
	dir := t.TempDir()
	libPath := filepath.Join(dir, "library")
	srcPath := filepath.Join(dir, "src.mkv")
	if err := os.WriteFile(srcPath, []byte("video"), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	meta := newTestMetaClient(map[string]metadata.Episode{
		"AAAA1111": {Arc: 1, Episode: 1, Title: "Romance Dawn 01", File: metadata.EpisodeFile{Version: "normal", CRC32: "AAAA1111"}},
	})
	store := newTestStore(t)
	acts := newTestActivityStore(t)
	tracker := downloads.NewTracker()
	parsed := &scanner.ParsedFilename{CRC32: "AAAA1111", Extension: "mkv"}

	err := importFile(srcPath, parsed, nil, "", "src.mkv", meta, store, acts, tracker, libPath, false)
	if err != nil {
		t.Fatalf("importFile: %v", err)
	}

	wantDst := filepath.Join(libPath, "01 - Romance Dawn", "S01E01 - Romance Dawn 01 [AAAA1111].mkv")
	if _, err := os.Stat(wantDst); err != nil {
		t.Errorf("expected imported file at %q: %v", wantDst, err)
	}
	if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
		t.Errorf("expected src to be moved away, stat err = %v", err)
	}

	store.Read(func(lib *library.Library) {
		arc, ok := lib.Arcs[1]
		if !ok {
			t.Fatal("expected arc 1 in library")
		}
		v := arc.Episodes["1"].Versions["normal"]
		if v.FilePath != wantDst {
			t.Errorf("library FilePath = %q, want %q", v.FilePath, wantDst)
		}
	})

	events := acts.All()
	if len(events) != 1 || events[0].Type != activity.EventImport {
		t.Errorf("expected exactly one import activity event, got %+v", events)
	}
}

func TestImportFile_AlreadyImportedDeletesTorrentWithoutMoving(t *testing.T) {
	dir := t.TempDir()
	libPath := filepath.Join(dir, "library")
	arcFolder := filepath.Join(libPath, "01 - Romance Dawn")
	if err := os.MkdirAll(arcFolder, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dst := filepath.Join(arcFolder, "S01E01 - Romance Dawn 01 [AAAA1111].mkv")
	if err := os.WriteFile(dst, []byte("already here"), 0644); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	srcPath := filepath.Join(dir, "src.mkv")
	if err := os.WriteFile(srcPath, []byte("video"), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	meta := newTestMetaClient(map[string]metadata.Episode{
		"AAAA1111": {Arc: 1, Episode: 1, Title: "Romance Dawn 01", File: metadata.EpisodeFile{Version: "normal", CRC32: "AAAA1111"}},
	})
	store := newTestStore(t)
	acts := newTestActivityStore(t)
	tracker := downloads.NewTracker()
	parsed := &scanner.ParsedFilename{CRC32: "AAAA1111", Extension: "mkv"}

	mock, qb := newMockQbit(t)
	err := importFile(srcPath, parsed, qb, "somehash", "src.mkv", meta, store, acts, tracker, libPath, true)
	if err != nil {
		t.Fatalf("importFile: %v", err)
	}

	// The already-present destination must be untouched, and the src left
	// alone — the whole point of this branch is "don't redo the move".
	b, _ := os.ReadFile(dst)
	if string(b) != "already here" {
		t.Errorf("destination file was overwritten: %q", b)
	}
	if _, err := os.Stat(srcPath); err != nil {
		t.Errorf("expected src to be left in place, got: %v", err)
	}
	if !mock.wasDeleted("somehash") {
		t.Error("expected the torrent to be deleted from qBittorrent")
	}
}

func TestImportFile_UnknownCRCReturnsError(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "src.mkv")
	os.WriteFile(srcPath, []byte("video"), 0644)

	meta := newTestMetaClient(map[string]metadata.Episode{})
	store := newTestStore(t)
	acts := newTestActivityStore(t)
	tracker := downloads.NewTracker()
	parsed := &scanner.ParsedFilename{CRC32: "DEADBEEF", Extension: "mkv"}

	err := importFile(srcPath, parsed, nil, "", "src.mkv", meta, store, acts, tracker, filepath.Join(dir, "lib"), false)
	if err == nil {
		t.Error("expected an error for an unrecognised CRC, got nil")
	}
	if _, err := os.Stat(srcPath); err != nil {
		t.Errorf("expected src to be left in place after a failed import: %v", err)
	}
}

// --- importTorrent ------------------------------------------------------

func TestImportTorrent_SingleFile(t *testing.T) {
	dir := t.TempDir()
	savePath := filepath.Join(dir, "downloads")
	libPath := filepath.Join(dir, "library")
	if err := os.MkdirAll(savePath, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	name := "[One Pace][1-2] Romance Dawn 01 [1080p][AAAA1111].mkv"
	if err := os.WriteFile(filepath.Join(savePath, name), []byte("video"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	meta := newTestMetaClient(map[string]metadata.Episode{
		"AAAA1111": {Arc: 1, Episode: 1, Title: "Romance Dawn 01", File: metadata.EpisodeFile{Version: "normal", CRC32: "AAAA1111"}},
	})
	store := newTestStore(t)
	acts := newTestActivityStore(t)
	tracker := downloads.NewTracker()
	mock, qb := newMockQbit(t)

	torrent := qbittorrent.TorrentInfo{Name: name, SavePath: savePath, Hash: "singlehash"}
	if err := importTorrent(torrent, qb, meta, store, acts, tracker, libPath, savePath); err != nil {
		t.Fatalf("importTorrent: %v", err)
	}

	wantDst := filepath.Join(libPath, "01 - Romance Dawn", "S01E01 - Romance Dawn 01 [AAAA1111].mkv")
	if _, err := os.Stat(wantDst); err != nil {
		t.Errorf("expected imported file at %q: %v", wantDst, err)
	}
	if !mock.wasDeleted("singlehash") {
		t.Error("expected the torrent to be deleted after a successful single-file import")
	}
}

// Arc-pack torrents contain multiple episode files under one folder; every
// recognised file should be imported independently and the torrent removed
// once nothing importable remains.
func TestImportTorrent_FolderWithMultipleEpisodes(t *testing.T) {
	dir := t.TempDir()
	contentPath := filepath.Join(dir, "downloads", "Gaimon Pack")
	libPath := filepath.Join(dir, "library")
	if err := os.MkdirAll(contentPath, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	files := map[string]string{
		"[One Pace][42,22] Gaimon 01 [1080p][AAAA1111].mkv": "AAAA1111",
		"[One Pace][23-24] Gaimon 02 [1080p][BBBB2222].mkv": "BBBB2222",
		"readme.txt": "",
	}
	for name := range files {
		if err := os.WriteFile(filepath.Join(contentPath, name), []byte("video"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	meta := newTestMetaClient(map[string]metadata.Episode{
		"AAAA1111": {Arc: 4, Episode: 1, Title: "Gaimon 01", File: metadata.EpisodeFile{Version: "normal", CRC32: "AAAA1111"}},
		"BBBB2222": {Arc: 4, Episode: 2, Title: "Gaimon 02", File: metadata.EpisodeFile{Version: "normal", CRC32: "BBBB2222"}},
	})
	meta.Cache.ArcsByNumber[4] = metadata.Arc{ArcNumber: 4, Title: "Gaimon"}
	store := newTestStore(t)
	acts := newTestActivityStore(t)
	tracker := downloads.NewTracker()
	mock, qb := newMockQbit(t)

	torrent := qbittorrent.TorrentInfo{Name: "Gaimon Pack", ContentPath: contentPath, Hash: "packhash"}
	if err := importTorrent(torrent, qb, meta, store, acts, tracker, libPath, ""); err != nil {
		t.Fatalf("importTorrent: %v", err)
	}

	for _, crc := range []string{"AAAA1111", "BBBB2222"} {
		found := false
		store.Read(func(lib *library.Library) {
			for _, v := range lib.Arcs[4].Episodes {
				for _, ver := range v.Versions {
					if ver.CRC32 == crc {
						found = true
					}
				}
			}
		})
		if !found {
			t.Errorf("expected an imported episode with CRC %s", crc)
		}
	}
	if !mock.wasDeleted("packhash") {
		t.Error("expected the folder torrent to be deleted once fully imported")
	}
}

func TestImportTorrent_FolderWithNoImportableFiles(t *testing.T) {
	dir := t.TempDir()
	contentPath := filepath.Join(dir, "downloads", "Junk")
	if err := os.MkdirAll(contentPath, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contentPath, "notes.txt"), []byte("hi"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	meta := newTestMetaClient(map[string]metadata.Episode{})
	store := newTestStore(t)
	acts := newTestActivityStore(t)
	tracker := downloads.NewTracker()
	mock, qb := newMockQbit(t)

	torrent := qbittorrent.TorrentInfo{Name: "Junk", ContentPath: contentPath, Hash: "junkhash"}
	if err := importTorrent(torrent, qb, meta, store, acts, tracker, filepath.Join(dir, "library"), ""); err != nil {
		t.Fatalf("importTorrent: %v", err)
	}
	if !mock.wasDeleted("junkhash") {
		t.Error("expected a torrent with no importable files to still be cleaned up")
	}
}

// --- poll -----------------------------------------------------------------

func TestPoll_MarksIncompleteTorrentsActiveWithoutImporting(t *testing.T) {
	dir := t.TempDir()
	meta := newTestMetaClient(map[string]metadata.Episode{
		"AAAA1111": {Arc: 1, Episode: 1, Title: "Romance Dawn 01", File: metadata.EpisodeFile{Version: "normal", CRC32: "AAAA1111"}},
	})
	store := newTestStore(t)
	acts := newTestActivityStore(t)
	tracker := downloads.NewTracker()
	mock, qb := newMockQbit(t)

	mock.setTorrents([]qbittorrent.TorrentInfo{
		{Name: "[One Pace][1-2] Romance Dawn 01 [1080p][AAAA1111].mkv", Hash: "h1", Progress: 0.5},
	})

	if err := poll(qb, meta, store, acts, tracker, filepath.Join(dir, "library"), dir); err != nil {
		t.Fatalf("poll: %v", err)
	}

	if !tracker.IsActive("AAAA1111") {
		t.Error("expected an in-progress torrent's CRC to be marked active")
	}
	if mock.wasDeleted("h1") {
		t.Error("an incomplete torrent should never be deleted")
	}
}

func TestPoll_ImportsCompletedTorrent(t *testing.T) {
	dir := t.TempDir()
	libPath := filepath.Join(dir, "library")
	name := "[One Pace][1-2] Romance Dawn 01 [1080p][AAAA1111].mkv"
	if err := os.WriteFile(filepath.Join(dir, name), []byte("video"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	meta := newTestMetaClient(map[string]metadata.Episode{
		"AAAA1111": {Arc: 1, Episode: 1, Title: "Romance Dawn 01", File: metadata.EpisodeFile{Version: "normal", CRC32: "AAAA1111"}},
	})
	store := newTestStore(t)
	acts := newTestActivityStore(t)
	tracker := downloads.NewTracker()
	mock, qb := newMockQbit(t)

	mock.setTorrents([]qbittorrent.TorrentInfo{
		{Name: name, SavePath: dir, Hash: "h1", Progress: 1},
	})

	if err := poll(qb, meta, store, acts, tracker, libPath, dir); err != nil {
		t.Fatalf("poll: %v", err)
	}

	// The actual import runs in a background goroutine (see poll's dedup
	// comment) — poll for the result instead of racing it with a fixed sleep.
	wantDst := filepath.Join(libPath, "01 - Romance Dawn", "S01E01 - Romance Dawn 01 [AAAA1111].mkv")
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(wantDst); err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, err := os.Stat(wantDst); err != nil {
		t.Fatalf("expected imported file at %q within timeout: %v", wantDst, err)
	}
	if !mock.wasDeleted("h1") {
		t.Error("expected the completed torrent to be deleted after import")
	}
}

func TestPoll_GetTorrentsError(t *testing.T) {
	_, qb := newMockQbit(t)
	qb.Host = "http://127.0.0.1:1" // nothing listening — connection refused

	meta := newTestMetaClient(map[string]metadata.Episode{})
	store := newTestStore(t)
	acts := newTestActivityStore(t)
	tracker := downloads.NewTracker()

	if err := poll(qb, meta, store, acts, tracker, t.TempDir(), t.TempDir()); err == nil {
		t.Error("expected poll to surface a GetTorrents error, got nil")
	}
}

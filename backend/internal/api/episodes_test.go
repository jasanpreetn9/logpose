package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"

	"onepace-library/internal/db"
	"onepace-library/internal/library"
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

// writeTempEpisodeFiles creates a video file plus its .nfo/-thumb.jpg
// sidecars in dir, and returns the video's path.
func writeTempEpisodeFiles(t *testing.T, dir, name string) string {
	t.Helper()
	videoPath := filepath.Join(dir, name+".mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0644); err != nil {
		t.Fatalf("write video: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+".nfo"), []byte("<nfo/>"), 0644); err != nil {
		t.Fatalf("write nfo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+"-thumb.jpg"), []byte("thumb"), 0644); err != nil {
		t.Fatalf("write thumb: %v", err)
	}
	return videoPath
}

func assertDeleted(t *testing.T, paths ...string) {
	t.Helper()
	for _, p := range paths {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("expected %s to be deleted, stat err = %v", p, err)
		}
	}
}

func TestHandleDeleteEpisodeVersion(t *testing.T) {
	store := newTestStore(t)
	dir := t.TempDir()
	videoPath := writeTempEpisodeFiles(t, dir, "S01E01 - Title [ABCD1234]")

	if err := store.Write(func(lib *library.Library) error {
		arc := lib.GetOrCreateArc(1, "Arc")
		arc.Episodes["1"] = library.Episode{
			EpisodeNumber: 1,
			Title:         "Title",
			Versions: map[string]library.EpisodeVersion{
				"normal": {CRC32: "ABCD1234", FilePath: videoPath, DownloadStatus: "imported"},
			},
		}
		return nil
	}); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	r := chi.NewRouter()
	r.Delete("/episodes/{crc}", HandleDeleteEpisodeVersion(store))

	req := httptest.NewRequest(http.MethodDelete, "/episodes/ABCD1234", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	assertDeleted(t,
		videoPath,
		filepath.Join(dir, "S01E01 - Title [ABCD1234].nfo"),
		filepath.Join(dir, "S01E01 - Title [ABCD1234]-thumb.jpg"),
	)

	store.Read(func(lib *library.Library) {
		if _, ok := lib.Arcs[1].Episodes["1"].Versions["normal"]; ok {
			t.Errorf("expected version entry to be removed from library")
		}
	})
}

func TestHandleDeleteEpisodeVersion_NotFound(t *testing.T) {
	store := newTestStore(t)

	r := chi.NewRouter()
	r.Delete("/episodes/{crc}", HandleDeleteEpisodeVersion(store))

	req := httptest.NewRequest(http.MethodDelete, "/episodes/DEADBEEF", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestHandleDeleteEpisode(t *testing.T) {
	store := newTestStore(t)
	dir := t.TempDir()
	normalPath := writeTempEpisodeFiles(t, dir, "S01E02 - Title [AAAA1111]")
	extendedPath := writeTempEpisodeFiles(t, dir, "S01E02 - Title Extended [BBBB2222]")

	if err := store.Write(func(lib *library.Library) error {
		arc := lib.GetOrCreateArc(1, "Arc")
		arc.Episodes["2"] = library.Episode{
			EpisodeNumber: 2,
			Title:         "Title",
			Versions: map[string]library.EpisodeVersion{
				"normal":   {CRC32: "AAAA1111", FilePath: normalPath, DownloadStatus: "imported"},
				"extended": {CRC32: "BBBB2222", FilePath: extendedPath, DownloadStatus: "imported"},
			},
		}
		return nil
	}); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	r := chi.NewRouter()
	r.Delete("/episodes/{arc}/{episode}", HandleDeleteEpisode(store))

	req := httptest.NewRequest(http.MethodDelete, "/episodes/1/2", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	assertDeleted(t, normalPath, extendedPath)

	store.Read(func(lib *library.Library) {
		if versions := lib.Arcs[1].Episodes["2"].Versions; len(versions) != 0 {
			t.Errorf("expected all versions removed, got %v", versions)
		}
	})
}

func TestHandleDeleteEpisode_NoOpWhenUntracked(t *testing.T) {
	store := newTestStore(t)

	r := chi.NewRouter()
	r.Delete("/episodes/{arc}/{episode}", HandleDeleteEpisode(store))

	req := httptest.NewRequest(http.MethodDelete, "/episodes/99/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (idempotent no-op); body=%s", rec.Code, rec.Body.String())
	}
}

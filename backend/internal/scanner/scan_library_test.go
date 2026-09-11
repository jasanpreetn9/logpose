package scanner

import (
	"os"
	"path/filepath"
	"testing"

	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
)

// newTestMetaClient builds a metadata.Client with its cache pre-populated,
// bypassing the network fetch in Refresh().
func newTestMetaClient(episodes map[string]metadata.Episode, arcs map[int]metadata.Arc) *metadata.Client {
	c := metadata.NewClient("", "")
	c.Cache.EpisodesByCRC = episodes
	c.Cache.ArcsByNumber = arcs
	return c
}

func TestScanLibrary_ImportsMatchingFile(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "S01E01 - Romance Dawn [AAAA1111].mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0644); err != nil {
		t.Fatalf("write video: %v", err)
	}

	meta := newTestMetaClient(
		map[string]metadata.Episode{
			"AAAA1111": {Arc: 1, Episode: 1, Title: "Romance Dawn", File: metadata.EpisodeFile{Version: "normal", CRC32: "AAAA1111"}},
		},
		map[int]metadata.Arc{1: {ArcNumber: 1, Title: "Romance Dawn"}},
	)
	lib := library.New()

	stats, err := ScanLibrary(dir, lib, meta)
	if err != nil {
		t.Fatalf("ScanLibrary: %v", err)
	}
	if stats.FilesFound != 1 {
		t.Errorf("FilesFound = %d, want 1", stats.FilesFound)
	}

	arc, ok := lib.Arcs[1]
	if !ok {
		t.Fatal("expected arc 1 to exist")
	}
	ep, ok := arc.Episodes["1"]
	if !ok {
		t.Fatal("expected episode 1 to exist")
	}
	if v := ep.Versions["normal"]; v.CRC32 != "AAAA1111" || v.FilePath != videoPath {
		t.Errorf("unexpected version: %+v", v)
	}
}

func TestScanLibrary_SkipsUnknownCRC(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "S01E01 - Title [DEADBEEF].mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0644); err != nil {
		t.Fatalf("write video: %v", err)
	}

	meta := newTestMetaClient(map[string]metadata.Episode{}, map[int]metadata.Arc{})
	lib := library.New()

	stats, err := ScanLibrary(dir, lib, meta)
	if err != nil {
		t.Fatalf("ScanLibrary: %v", err)
	}
	if stats.FilesFound != 0 {
		t.Errorf("FilesFound = %d, want 0 for an unrecognised CRC", stats.FilesFound)
	}
	if len(lib.Arcs) != 0 {
		t.Errorf("expected no arcs to be created, got %v", lib.Arcs)
	}
}

func TestScanLibrary_SkipsNonVideoFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "S01E01 - Title [AAAA1111].nfo"), []byte("<nfo/>"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	meta := newTestMetaClient(
		map[string]metadata.Episode{"AAAA1111": {Arc: 1, Episode: 1, Title: "Title"}},
		map[int]metadata.Arc{},
	)
	lib := library.New()

	stats, err := ScanLibrary(dir, lib, meta)
	if err != nil {
		t.Fatalf("ScanLibrary: %v", err)
	}
	if stats.FilesFound != 0 {
		t.Errorf("FilesFound = %d, want 0", stats.FilesFound)
	}
}

// A version whose file has disappeared from disk since the last scan must be
// marked "missing" with its path cleared, so it shows up as re-downloadable
// rather than silently still pointing at a dead path.
func TestScanLibrary_MarksDeletedFileAsMissing(t *testing.T) {
	dir := t.TempDir()
	meta := newTestMetaClient(map[string]metadata.Episode{}, map[int]metadata.Arc{})
	lib := library.New()

	arc := lib.GetOrCreateArc(1, "Arc")
	arc.Episodes["1"] = library.Episode{
		EpisodeNumber: 1,
		Versions: map[string]library.EpisodeVersion{
			"normal": {CRC32: "AAAA1111", FilePath: filepath.Join(dir, "gone.mkv"), DownloadStatus: "imported"},
		},
	}

	stats, err := ScanLibrary(dir, lib, meta)
	if err != nil {
		t.Fatalf("ScanLibrary: %v", err)
	}
	if stats.FilesMarkedMissing != 1 {
		t.Errorf("FilesMarkedMissing = %d, want 1", stats.FilesMarkedMissing)
	}
	v := lib.Arcs[1].Episodes["1"].Versions["normal"]
	if v.DownloadStatus != "missing" || v.FilePath != "" {
		t.Errorf("unexpected version after scan: %+v", v)
	}
}

// A version whose file is still present on disk must not be touched, even
// when other versions of other episodes were marked missing.
func TestScanLibrary_LeavesPresentFilesAlone(t *testing.T) {
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "S01E01 - Title [AAAA1111].mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0644); err != nil {
		t.Fatalf("write video: %v", err)
	}

	meta := newTestMetaClient(
		map[string]metadata.Episode{"AAAA1111": {Arc: 1, Episode: 1, Title: "Title", File: metadata.EpisodeFile{Version: "normal", CRC32: "AAAA1111"}}},
		map[int]metadata.Arc{},
	)
	lib := library.New()
	arc := lib.GetOrCreateArc(1, "Arc")
	arc.Episodes["1"] = library.Episode{
		EpisodeNumber: 1,
		Versions: map[string]library.EpisodeVersion{
			"normal": {CRC32: "AAAA1111", FilePath: videoPath, DownloadStatus: "imported"},
		},
	}

	stats, err := ScanLibrary(dir, lib, meta)
	if err != nil {
		t.Fatalf("ScanLibrary: %v", err)
	}
	if stats.FilesMarkedMissing != 0 {
		t.Errorf("FilesMarkedMissing = %d, want 0", stats.FilesMarkedMissing)
	}
	v := lib.Arcs[1].Episodes["1"].Versions["normal"]
	if v.DownloadStatus != "imported" || v.FilePath != videoPath {
		t.Errorf("present file was incorrectly modified: %+v", v)
	}
}

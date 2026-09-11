package scanner

import (
	"testing"

	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
)

func TestAddOrUpdateEpisode_CreatesArcAndEpisode(t *testing.T) {
	lib := library.New()
	parsed := &ParsedFilename{ArcNumber: 1, EpisodeNum: 1, CRC32: "AAAA1111"}
	meta := metadata.Episode{
		Arc: 1, Episode: 1, Title: "Romance Dawn 01", Description: "desc",
		File: metadata.EpisodeFile{Version: "normal", CRC32: "AAAA1111"},
	}

	entry := AddOrUpdateEpisode(lib, "/lib/S01E01.mkv", parsed, meta, "Romance Dawn")

	if entry.Title != "Romance Dawn 01" {
		t.Errorf("Title = %q, want %q", entry.Title, "Romance Dawn 01")
	}
	arc, ok := lib.Arcs[1]
	if !ok {
		t.Fatal("expected arc 1 to be created")
	}
	if arc.Title != "Romance Dawn" {
		t.Errorf("arc title = %q, want %q", arc.Title, "Romance Dawn")
	}
	ep, ok := arc.Episodes["1"]
	if !ok {
		t.Fatal("expected episode key \"1\" to exist")
	}
	v, ok := ep.Versions["normal"]
	if !ok {
		t.Fatal("expected \"normal\" version to exist")
	}
	if v.CRC32 != "AAAA1111" || v.FilePath != "/lib/S01E01.mkv" || v.DownloadStatus != "imported" {
		t.Errorf("unexpected version: %+v", v)
	}
}

// Download-format filenames never carry an arc number (see filenames_test.go);
// merge must fall back to the metadata's arc number in that case.
func TestAddOrUpdateEpisode_FallsBackToMetadataArcNumber(t *testing.T) {
	lib := library.New()
	parsed := &ParsedFilename{ArcNumber: 0, EpisodeNum: 1, CRC32: "AAAA1111"}
	meta := metadata.Episode{
		Arc: 7, Episode: 1, Title: "Title",
		File: metadata.EpisodeFile{Version: "normal", CRC32: "AAAA1111"},
	}

	AddOrUpdateEpisode(lib, "/lib/S07E01.mkv", parsed, meta, "Arc Seven")

	if _, ok := lib.Arcs[7]; !ok {
		t.Fatalf("expected arc 7 (from metadata), got arcs: %v", lib.Arcs)
	}
}

// Importing a second version (e.g. "extended") of an episode that already
// has a "normal" version imported must not clobber the existing version.
func TestAddOrUpdateEpisode_AddsVersionWithoutClobberingExisting(t *testing.T) {
	lib := library.New()
	normalParsed := &ParsedFilename{ArcNumber: 1, EpisodeNum: 1, CRC32: "AAAA1111"}
	normalMeta := metadata.Episode{
		Arc: 1, Episode: 1, Title: "Title",
		File: metadata.EpisodeFile{Version: "normal", CRC32: "AAAA1111"},
	}
	AddOrUpdateEpisode(lib, "/lib/normal.mkv", normalParsed, normalMeta, "Arc")

	extendedParsed := &ParsedFilename{ArcNumber: 1, EpisodeNum: 1, CRC32: "BBBB2222"}
	extendedMeta := metadata.Episode{
		Arc: 1, Episode: 1, Title: "Title",
		File: metadata.EpisodeFile{Version: "extended", CRC32: "BBBB2222"},
	}
	entry := AddOrUpdateEpisode(lib, "/lib/extended.mkv", extendedParsed, extendedMeta, "Arc")

	if len(entry.Versions) != 2 {
		t.Fatalf("expected 2 versions, got %d: %+v", len(entry.Versions), entry.Versions)
	}
	if v := entry.Versions["normal"]; v.FilePath != "/lib/normal.mkv" {
		t.Errorf("normal version was clobbered: %+v", v)
	}
	if v := entry.Versions["extended"]; v.FilePath != "/lib/extended.mkv" {
		t.Errorf("extended version missing/wrong: %+v", v)
	}
}

// Re-importing the same version (e.g. a re-scan after moving the file) must
// update title/description and the version's file info, not duplicate it.
func TestAddOrUpdateEpisode_UpdatesExistingVersion(t *testing.T) {
	lib := library.New()
	parsed := &ParsedFilename{ArcNumber: 1, EpisodeNum: 1, CRC32: "AAAA1111"}
	meta := metadata.Episode{
		Arc: 1, Episode: 1, Title: "Old Title",
		File: metadata.EpisodeFile{Version: "normal", CRC32: "AAAA1111"},
	}
	AddOrUpdateEpisode(lib, "/lib/old-path.mkv", parsed, meta, "Arc")

	meta.Title = "New Title"
	entry := AddOrUpdateEpisode(lib, "/lib/new-path.mkv", parsed, meta, "Arc")

	if entry.Title != "New Title" {
		t.Errorf("Title = %q, want %q", entry.Title, "New Title")
	}
	if len(entry.Versions) != 1 {
		t.Fatalf("expected 1 version (updated in place), got %d", len(entry.Versions))
	}
	if v := entry.Versions["normal"]; v.FilePath != "/lib/new-path.mkv" {
		t.Errorf("FilePath = %q, want updated path", v.FilePath)
	}
}

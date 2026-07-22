package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func openMem(t *testing.T) *DB {
	t.Helper()
	d, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open(:memory:): %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestOpenDB(t *testing.T) {
	// Schema test uses in-memory DB (fast).
	d := openMem(t)
	for _, tbl := range []string{"arcs", "episodes", "events"} {
		var name string
		err := d.SQL.QueryRow(
			`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, tbl,
		).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", tbl, err)
		}
	}

	// WAL mode requires a file-based DB.
	dir := t.TempDir()
	fd, err := Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("Open(file): %v", err)
	}
	defer fd.Close()

	var mode string
	if err := fd.SQL.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatalf("journal_mode query: %v", err)
	}
	if mode != "wal" {
		t.Errorf("expected journal_mode=wal, got %q", mode)
	}
	_ = os.Remove(filepath.Join(dir, "test.db"))
}

func TestInsertEvent(t *testing.T) {
	d := openMem(t)

	if err := d.InsertEvent("import", "Imported: ep1", "/library/ep1.mkv", true); err != nil {
		t.Fatalf("InsertEvent: %v", err)
	}

	events, err := d.ListEvents(10)
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	ev := events[0]
	if ev.Type != "import" {
		t.Errorf("Type: want %q, got %q", "import", ev.Type)
	}
	if ev.Message != "Imported: ep1" {
		t.Errorf("Message: want %q, got %q", "Imported: ep1", ev.Message)
	}
	if ev.Payload != "/library/ep1.mkv" {
		t.Errorf("Payload: want %q, got %q", "/library/ep1.mkv", ev.Payload)
	}
	if !ev.Success {
		t.Errorf("Success: want true, got false")
	}
	if ev.CreatedAt.IsZero() {
		t.Errorf("CreatedAt should not be zero")
	}
}

func TestUpsertEpisode(t *testing.T) {
	d := openMem(t)

	row := EpisodeRow{
		ArcNumber:     1,
		EpisodeNumber: 1,
		Title:         "The Beginning",
		Description:   "First ep",
		Monitored:     true,
		Versions: map[string]VersionRow{
			"normal": {CRC32: "AABBCCDD", FilePath: "", DownloadStatus: "missing"},
		},
	}

	if err := d.Tx(func(tx *sql.Tx) error {
		return d.UpsertEpisode(tx, row)
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	// Update the file path via a second upsert (same version).
	row.Versions["normal"] = VersionRow{CRC32: "AABBCCDD", FilePath: "/library/arc1/ep1.mkv", DownloadStatus: "imported"}
	if err := d.Tx(func(tx *sql.Tx) error {
		return d.UpsertEpisode(tx, row)
	}); err != nil {
		t.Fatalf("update upsert: %v", err)
	}

	// Only one row should exist.
	var count int
	if err := d.SQL.QueryRow(`SELECT COUNT(*) FROM episodes WHERE arc_number=1 AND episode_number=1`).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 row after upsert, got %d", count)
	}

	// The updated values should be persisted.
	all, err := d.GetAllEpisodes()
	if err != nil {
		t.Fatalf("GetAllEpisodes: %v", err)
	}
	got, ok := all[1]["1"]
	if !ok {
		t.Fatalf("episode arc=1 ep=1 not found after upsert")
	}
	normal, ok := got.Versions["normal"]
	if !ok {
		t.Fatalf("version %q not found", "normal")
	}
	if normal.FilePath != "/library/arc1/ep1.mkv" {
		t.Errorf("FilePath: want %q, got %q", "/library/arc1/ep1.mkv", normal.FilePath)
	}
	if normal.DownloadStatus != "imported" {
		t.Errorf("DownloadStatus: want %q, got %q", "imported", normal.DownloadStatus)
	}
}

// TestUpsertEpisodeMultipleVersions is a regression test for the bug where
// importing a second version (e.g. "extended") of an episode overwrote the
// first version's row, causing the UI to show a perpetual "upgrade
// available" ping-pong between the two versions.
func TestUpsertEpisodeMultipleVersions(t *testing.T) {
	d := openMem(t)

	normalRow := EpisodeRow{
		ArcNumber: 1, EpisodeNumber: 1, Title: "The Beginning", Monitored: true,
		Versions: map[string]VersionRow{
			"normal": {CRC32: "AAAA1111", FilePath: "/library/arc1/ep1-normal.mkv", DownloadStatus: "imported"},
		},
	}
	if err := d.Tx(func(tx *sql.Tx) error { return d.UpsertEpisode(tx, normalRow) }); err != nil {
		t.Fatalf("upsert normal: %v", err)
	}

	// Now import the extended version — this must NOT erase the normal version's row.
	extendedRow := EpisodeRow{
		ArcNumber: 1, EpisodeNumber: 1, Title: "The Beginning", Monitored: true,
		Versions: map[string]VersionRow{
			"normal":   {CRC32: "AAAA1111", FilePath: "/library/arc1/ep1-normal.mkv", DownloadStatus: "imported"},
			"extended": {CRC32: "BBBB2222", FilePath: "/library/arc1/ep1-extended.mkv", DownloadStatus: "imported"},
		},
	}
	if err := d.Tx(func(tx *sql.Tx) error { return d.UpsertEpisode(tx, extendedRow) }); err != nil {
		t.Fatalf("upsert extended: %v", err)
	}

	var count int
	if err := d.SQL.QueryRow(`SELECT COUNT(*) FROM episodes WHERE arc_number=1 AND episode_number=1`).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 row (one episode, multiple versions), got %d", count)
	}

	all, err := d.GetAllEpisodes()
	if err != nil {
		t.Fatalf("GetAllEpisodes: %v", err)
	}
	got, ok := all[1]["1"]
	if !ok {
		t.Fatalf("episode arc=1 ep=1 not found")
	}
	if len(got.Versions) != 2 {
		t.Fatalf("expected 2 versions tracked, got %d: %+v", len(got.Versions), got.Versions)
	}
	if v, ok := got.Versions["normal"]; !ok || v.DownloadStatus != "imported" || v.CRC32 != "AAAA1111" {
		t.Errorf("normal version not preserved after importing extended: %+v (ok=%v)", v, ok)
	}
	if v, ok := got.Versions["extended"]; !ok || v.DownloadStatus != "imported" || v.CRC32 != "BBBB2222" {
		t.Errorf("extended version not recorded: %+v (ok=%v)", v, ok)
	}
}

// TestMigrateVersionsColumnBackfill verifies that a row written before
// versions_json existed (flat crc32/version/file_path/download_status only)
// gets correctly folded into versions_json by the migration, so upgrading
// doesn't lose track of already-imported files.
func TestMigrateVersionsColumnBackfill(t *testing.T) {
	d := openMem(t)

	if _, err := d.SQL.Exec(`
		INSERT INTO episodes(arc_number,episode_number,crc32,version,file_path,download_status,title,monitored)
		VALUES(5,10,'DEADBEEF','extended','/library/arc5/ep10-ext.mkv','imported','Old Row',1)
	`); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	if err := d.migrateVersionsColumn(); err != nil {
		t.Fatalf("migrateVersionsColumn: %v", err)
	}

	all, err := d.GetAllEpisodes()
	if err != nil {
		t.Fatalf("GetAllEpisodes: %v", err)
	}
	got, ok := all[5]["10"]
	if !ok {
		t.Fatalf("episode arc=5 ep=10 not found after backfill")
	}
	v, ok := got.Versions["extended"]
	if !ok {
		t.Fatalf("expected backfilled 'extended' version, got %+v", got.Versions)
	}
	if v.CRC32 != "DEADBEEF" || v.FilePath != "/library/arc5/ep10-ext.mkv" || v.DownloadStatus != "imported" {
		t.Errorf("backfilled version mismatch: %+v", v)
	}
}

func TestLibraryRoundTrip(t *testing.T) {
	d := openMem(t)

	type arcSpec struct {
		num      int
		title    string
		episodes []EpisodeRow
	}

	specs := []arcSpec{
		{
			num:   1,
			title: "Romance Dawn",
			episodes: []EpisodeRow{
				{ArcNumber: 1, EpisodeNumber: 1, Title: "Ep 1-1", Monitored: true, Versions: map[string]VersionRow{"normal": {CRC32: "AAA1", DownloadStatus: "missing"}}},
				{ArcNumber: 1, EpisodeNumber: 2, Title: "Ep 1-2", Monitored: true, Versions: map[string]VersionRow{"normal": {CRC32: "AAA2", DownloadStatus: "missing"}}},
				{ArcNumber: 1, EpisodeNumber: 3, Title: "Ep 1-3", Monitored: true, Versions: map[string]VersionRow{"normal": {CRC32: "AAA3", DownloadStatus: "missing"}}},
			},
		},
		{
			num:   2,
			title: "Orange Town",
			episodes: []EpisodeRow{
				{ArcNumber: 2, EpisodeNumber: 1, Title: "Ep 2-1", Monitored: true, Versions: map[string]VersionRow{"normal": {CRC32: "BBB1", DownloadStatus: "missing"}}},
				{ArcNumber: 2, EpisodeNumber: 2, Title: "Ep 2-2", Monitored: false, Versions: map[string]VersionRow{"normal": {CRC32: "BBB2", DownloadStatus: "missing"}}},
				{ArcNumber: 2, EpisodeNumber: 3, Title: "Ep 2-3", Monitored: true, Versions: map[string]VersionRow{"normal": {CRC32: "BBB3", DownloadStatus: "missing"}}},
			},
		},
	}

	if err := d.Tx(func(tx *sql.Tx) error {
		for _, spec := range specs {
			if err := d.SaveArc(tx, spec.num, spec.title, true); err != nil {
				return err
			}
			for _, ep := range spec.episodes {
				if err := d.UpsertEpisode(tx, ep); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("persist: %v", err)
	}

	arcs, err := d.GetAllArcs()
	if err != nil {
		t.Fatalf("GetAllArcs: %v", err)
	}
	if len(arcs) != 2 {
		t.Errorf("expected 2 arcs, got %d", len(arcs))
	}

	episodes, err := d.GetAllEpisodes()
	if err != nil {
		t.Fatalf("GetAllEpisodes: %v", err)
	}
	if len(episodes) != 2 {
		t.Errorf("expected episodes for 2 arcs, got %d", len(episodes))
	}
	for _, spec := range specs {
		arcEps, ok := episodes[spec.num]
		if !ok {
			t.Errorf("arc %d not found in episodes map", spec.num)
			continue
		}
		if len(arcEps) != len(spec.episodes) {
			t.Errorf("arc %d: expected %d episodes, got %d", spec.num, len(spec.episodes), len(arcEps))
		}
		for _, want := range spec.episodes {
			key := "1"
			if want.EpisodeNumber > 1 {
				key = string(rune('0' + want.EpisodeNumber))
			}
			got, ok := arcEps[key]
			if !ok {
				t.Errorf("arc %d ep %d not found", spec.num, want.EpisodeNumber)
				continue
			}
			wantVersion := want.Versions["normal"]
			gotVersion := got.Versions["normal"]
			if gotVersion.CRC32 != wantVersion.CRC32 {
				t.Errorf("arc %d ep %d: CRC32 want %q got %q", spec.num, want.EpisodeNumber, wantVersion.CRC32, gotVersion.CRC32)
			}
			if got.Monitored != want.Monitored {
				t.Errorf("arc %d ep %d: Monitored want %v got %v", spec.num, want.EpisodeNumber, want.Monitored, got.Monitored)
			}
		}
	}
}

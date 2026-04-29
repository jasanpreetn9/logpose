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
		ArcNumber:      1,
		EpisodeNumber:  1,
		CRC32:          "AABBCCDD",
		Version:        "normal",
		FilePath:       "",
		Title:          "The Beginning",
		Description:    "First ep",
		DownloadStatus: "missing",
		Monitored:      true,
		LastChecked:    "",
	}

	if err := d.Tx(func(tx *sql.Tx) error {
		return d.UpsertEpisode(tx, row)
	}); err != nil {
		t.Fatalf("initial upsert: %v", err)
	}

	// Update the file path via a second upsert.
	row.FilePath = "/library/arc1/ep1.mkv"
	row.DownloadStatus = "imported"
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
	if got.FilePath != "/library/arc1/ep1.mkv" {
		t.Errorf("FilePath: want %q, got %q", "/library/arc1/ep1.mkv", got.FilePath)
	}
	if got.DownloadStatus != "imported" {
		t.Errorf("DownloadStatus: want %q, got %q", "imported", got.DownloadStatus)
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
				{ArcNumber: 1, EpisodeNumber: 1, CRC32: "AAA1", Version: "normal", Title: "Ep 1-1", DownloadStatus: "missing", Monitored: true},
				{ArcNumber: 1, EpisodeNumber: 2, CRC32: "AAA2", Version: "normal", Title: "Ep 1-2", DownloadStatus: "missing", Monitored: true},
				{ArcNumber: 1, EpisodeNumber: 3, CRC32: "AAA3", Version: "normal", Title: "Ep 1-3", DownloadStatus: "missing", Monitored: true},
			},
		},
		{
			num:   2,
			title: "Orange Town",
			episodes: []EpisodeRow{
				{ArcNumber: 2, EpisodeNumber: 1, CRC32: "BBB1", Version: "normal", Title: "Ep 2-1", DownloadStatus: "missing", Monitored: true},
				{ArcNumber: 2, EpisodeNumber: 2, CRC32: "BBB2", Version: "normal", Title: "Ep 2-2", DownloadStatus: "missing", Monitored: false},
				{ArcNumber: 2, EpisodeNumber: 3, CRC32: "BBB3", Version: "normal", Title: "Ep 2-3", DownloadStatus: "missing", Monitored: true},
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
			if got.CRC32 != want.CRC32 {
				t.Errorf("arc %d ep %d: CRC32 want %q got %q", spec.num, want.EpisodeNumber, want.CRC32, got.CRC32)
			}
			if got.Monitored != want.Monitored {
				t.Errorf("arc %d ep %d: Monitored want %v got %v", spec.num, want.EpisodeNumber, want.Monitored, got.Monitored)
			}
		}
	}
}

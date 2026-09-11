package scanner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
)

func TestScanDownloads_ImportsAndRenamesFile(t *testing.T) {
	downloadDir := t.TempDir()
	libraryDir := t.TempDir()

	src := filepath.Join(downloadDir, "[One Pace][1-2] Romance Dawn 01 [1080p][AAAA1111].mkv")
	if err := os.WriteFile(src, []byte("video"), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	meta := newTestMetaClient(
		map[string]metadata.Episode{
			"AAAA1111": {Arc: 1, Episode: 1, Title: "Romance Dawn", File: metadata.EpisodeFile{Version: "normal", CRC32: "AAAA1111"}},
		},
		map[int]metadata.Arc{1: {ArcNumber: 1, Title: "Romance Dawn"}},
	)
	lib := library.New()

	imported, err := ScanDownloads(downloadDir, libraryDir, lib, meta)
	if err != nil {
		t.Fatalf("ScanDownloads: %v", err)
	}
	if len(imported) != 1 {
		t.Fatalf("expected 1 imported file, got %d: %+v", len(imported), imported)
	}
	if imported[0].Title != "Romance Dawn" {
		t.Errorf("Title = %q, want %q", imported[0].Title, "Romance Dawn")
	}

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("expected source file to be moved out of downloads, stat err = %v", err)
	}
	wantDst := filepath.Join(libraryDir, "01 - Romance Dawn", "S01E01 - Romance Dawn [AAAA1111].mkv")
	if imported[0].DstPath != wantDst {
		t.Errorf("DstPath = %q, want %q", imported[0].DstPath, wantDst)
	}
	if _, err := os.Stat(wantDst); err != nil {
		t.Errorf("expected file at %q: %v", wantDst, err)
	}

	ep := lib.Arcs[1].Episodes["1"]
	if v := ep.Versions["normal"]; v.FilePath != wantDst {
		t.Errorf("library not updated with new path: %+v", v)
	}
}

func TestScanDownloads_SkipsUnknownCRC(t *testing.T) {
	downloadDir := t.TempDir()
	libraryDir := t.TempDir()
	src := filepath.Join(downloadDir, "[One Pace][1-2] Romance Dawn 01 [1080p][DEADBEEF].mkv")
	if err := os.WriteFile(src, []byte("video"), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	meta := newTestMetaClient(map[string]metadata.Episode{}, map[int]metadata.Arc{})
	lib := library.New()

	imported, err := ScanDownloads(downloadDir, libraryDir, lib, meta)
	if err != nil {
		t.Fatalf("ScanDownloads: %v", err)
	}
	if len(imported) != 0 {
		t.Fatalf("expected nothing imported, got %+v", imported)
	}
	if _, err := os.Stat(src); err != nil {
		t.Errorf("expected unmatched file to be left in place: %v", err)
	}
}

// Regression test for #10/#11: a file still being written (its name already
// matches a known episode, but its size is still growing) must be left
// alone by a downloads scan rather than imported mid-write.
func TestScanDownloads_SkipsStillGrowingFile(t *testing.T) {
	downloadDir := t.TempDir()
	libraryDir := t.TempDir()
	src := filepath.Join(downloadDir, "[One Pace][1-2] Romance Dawn 01 [1080p][AAAA1111].mkv")
	if err := os.WriteFile(src, []byte("partial"), 0644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	meta := newTestMetaClient(
		map[string]metadata.Episode{
			"AAAA1111": {Arc: 1, Episode: 1, Title: "Romance Dawn", File: metadata.EpisodeFile{Version: "normal", CRC32: "AAAA1111"}},
		},
		map[int]metadata.Arc{},
	)
	lib := library.New()

	// Simulate the torrent client still appending to the file partway
	// through ScanDownloads' stability check.
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			case <-time.After(fileStabilityWait / 4):
				f, err := os.OpenFile(src, os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					return
				}
				f.WriteString("x")
				f.Close()
			}
		}
	}()

	imported, err := ScanDownloads(downloadDir, libraryDir, lib, meta)
	close(stop)
	<-done

	if err != nil {
		t.Fatalf("ScanDownloads: %v", err)
	}
	if len(imported) != 0 {
		t.Fatalf("expected the still-growing file to be skipped, got %+v", imported)
	}
	if _, err := os.Stat(src); err != nil {
		t.Errorf("expected source file to remain in downloads: %v", err)
	}
	if len(lib.Arcs) != 0 {
		t.Errorf("expected no library entry for an unstable file, got %v", lib.Arcs)
	}
}

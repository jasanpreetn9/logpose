package nfo

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
)

func TestGenerateEpisodeNFO_ExtendedTitleTag(t *testing.T) {
	ep := library.Episode{Title: "New Emperors", Description: "desc"}
	version := library.EpisodeVersion{CRC32: "ABCD1234"}

	cases := []struct {
		version   string
		wantTitle string
	}{
		{"extended", "New Emperors (Extended)"},
		{"normal", "New Emperors"},
	}

	for _, c := range cases {
		meta := metadata.Episode{
			Arc: 36, Episode: 1, Released: "2024-01-01",
			File: metadata.EpisodeFile{Version: c.version},
		}
		out := filepath.Join(t.TempDir(), "episode.nfo")
		if err := GenerateEpisodeNFO(ep, version, meta, "Egghead", out); err != nil {
			t.Fatalf("GenerateEpisodeNFO(%s): %v", c.version, err)
		}
		b, err := os.ReadFile(out)
		if err != nil {
			t.Fatalf("read nfo: %v", err)
		}
		var got EpisodeNFO
		if err := xml.Unmarshal(b, &got); err != nil {
			t.Fatalf("unmarshal nfo: %v", err)
		}
		if got.Title != c.wantTitle {
			t.Errorf("version=%s: title = %q, want %q", c.version, got.Title, c.wantTitle)
		}
	}
}

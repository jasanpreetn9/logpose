package scanner

import "testing"

func TestParseOnePaceFilename_Download(t *testing.T) {
	p, err := ParseOnePaceFilename("[One Pace][1058-1059] Egghead 01 [1080p][CA3F14A8].mkv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Mode != "download" {
		t.Errorf("Mode = %q, want %q", p.Mode, "download")
	}
	if p.AnimeStart != 1058 || p.AnimeEnd != 1059 {
		t.Errorf("AnimeStart/End = %d/%d, want 1058/1059", p.AnimeStart, p.AnimeEnd)
	}
	if p.ArcTitle != "Egghead" {
		t.Errorf("ArcTitle = %q, want %q", p.ArcTitle, "Egghead")
	}
	if p.EpisodeNum != 1 {
		t.Errorf("EpisodeNum = %d, want 1", p.EpisodeNum)
	}
	if p.Resolution != "1080p" {
		t.Errorf("Resolution = %q, want %q", p.Resolution, "1080p")
	}
	if p.CRC32 != "CA3F14A8" {
		t.Errorf("CRC32 = %q, want %q", p.CRC32, "CA3F14A8")
	}
	if p.Extension != "mkv" {
		t.Errorf("Extension = %q, want %q", p.Extension, "mkv")
	}
	// Download-format filenames carry no arc number — merge.go falls back
	// to metadata for that.
	if p.ArcNumber != 0 {
		t.Errorf("ArcNumber = %d, want 0 (unset for download format)", p.ArcNumber)
	}
}

// Regression test for #2: comma-separated anime episode lists (arc-pack
// releases spanning non-contiguous anime episodes) failed to parse before
// the fix in PR #3.
func TestParseOnePaceFilename_Download_CommaSeparatedEpisodes(t *testing.T) {
	p, err := ParseOnePaceFilename("[One Pace][42,22] Gaimon 01 [1080p][9269F40F].mkv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.AnimeStart != 42 {
		t.Errorf("AnimeStart = %d, want 42", p.AnimeStart)
	}
	if p.ArcTitle != "Gaimon" {
		t.Errorf("ArcTitle = %q, want %q", p.ArcTitle, "Gaimon")
	}
	if p.CRC32 != "9269F40F" {
		t.Errorf("CRC32 = %q, want %q", p.CRC32, "9269F40F")
	}
}

func TestParseOnePaceFilename_Download_ExtendedVariants(t *testing.T) {
	cases := []string{
		"[One Pace][1-2] Romance Dawn 01 (Extended) [1080p][12345678].mkv",
		"[One Pace][1-2] Romance Dawn 01 (EXT) [1080p][12345678].mkv",
		"[One Pace][1-2] Romance Dawn 01 EXTENDED [1080p][12345678].mkv",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			p, err := ParseOnePaceFilename(name)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.EpisodeNum != 1 {
				t.Errorf("EpisodeNum = %d, want 1", p.EpisodeNum)
			}
			if p.ArcTitle != "Romance Dawn" {
				t.Errorf("ArcTitle = %q, want %q", p.ArcTitle, "Romance Dawn")
			}
		})
	}
}

func TestParseOnePaceFilename_Download_Mp4(t *testing.T) {
	p, err := ParseOnePaceFilename("[One Pace][1-2] Romance Dawn 01 [1080p][12345678].mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Extension != "mp4" {
		t.Errorf("Extension = %q, want %q", p.Extension, "mp4")
	}
}

func TestParseOnePaceFilename_Library(t *testing.T) {
	p, err := ParseOnePaceFilename("S36E01 - New Emperors [CA3F14A8].mkv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Mode != "library" {
		t.Errorf("Mode = %q, want %q", p.Mode, "library")
	}
	if p.ArcNumber != 36 {
		t.Errorf("ArcNumber = %d, want 36", p.ArcNumber)
	}
	if p.EpisodeNum != 1 {
		t.Errorf("EpisodeNum = %d, want 1", p.EpisodeNum)
	}
	if p.CRC32 != "CA3F14A8" {
		t.Errorf("CRC32 = %q, want %q", p.CRC32, "CA3F14A8")
	}
}

func TestParseOnePaceFilename_CRC32Uppercased(t *testing.T) {
	p, err := ParseOnePaceFilename("S01E01 - Title [ca3f14a8].mkv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.CRC32 != "CA3F14A8" {
		t.Errorf("CRC32 = %q, want uppercased %q", p.CRC32, "CA3F14A8")
	}
}

func TestParseOnePaceFilename_Invalid(t *testing.T) {
	cases := []string{
		"random_video.mkv",
		"[One Pace] Egghead 01.mkv",                             // missing episode range/CRC
		"S01E01 - Title.mkv",                                    // missing CRC
		"S01E01 - Title [ZZZZZZZZ].mkv",                         // non-hex "CRC"
		"[One Pace][1-2] Romance Dawn 01 [1080p][12345678].avi", // unsupported extension
		"",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseOnePaceFilename(name); err == nil {
				t.Errorf("expected error for %q, got nil", name)
			}
		})
	}
}

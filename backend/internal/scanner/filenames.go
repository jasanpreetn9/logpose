package scanner

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type ParsedFilename struct {
	// From download filenames
	AnimeStart int
	AnimeEnd   int
	ArcTitle   string
	Resolution string

	// Common fields
	ArcNumber  int
	EpisodeNum int
	CRC32      string
	Extension  string
	Mode       string // "download" or "library"
}

// Download format: [One Pace][1058-1059] Egghead 01 [1080p][CA3F14A8].mkv
var downloadFilenameRegex = regexp.MustCompile(
	`^\[One Pace\]\[(\d+)(?:-(\d+))?\]\s(.+?)\s(\d+)(?:\s(?:\(?(?:Extended|EXT|EXTENDED)\)?))?\s\[(\d+p)\]\[([A-Fa-f0-9]{8})\]\.(mkv|mp4)$`,
)

// Library format: S36E01 - New Emperors [CA3F14A8].mkv
var libraryFilenameRegex = regexp.MustCompile(
	`^S(\d+)E(\d+)\s-\s.*\s\[([A-Fa-f0-9]{8})\]\.(mkv|mp4)$`,
)

func ParseOnePaceFilename(filename string) (*ParsedFilename, error) {

	// Try DOWNLOAD FORMAT
	if m := downloadFilenameRegex.FindStringSubmatch(filename); m != nil {
		animeStart, _ := strconv.Atoi(m[1])
		animeEnd, _ := strconv.Atoi(m[2])
		episodeNum, _ := strconv.Atoi(m[4])

		return &ParsedFilename{
			AnimeStart: animeStart,
			AnimeEnd:   animeEnd,
			ArcTitle:   m[3],
			EpisodeNum: episodeNum,
			Resolution: m[5],
			CRC32:      strings.ToUpper(m[6]),
			Extension:  m[7],
			Mode:       "download",
		}, nil
	}

	// Try LIBRARY FORMAT
	if m := libraryFilenameRegex.FindStringSubmatch(filename); m != nil {
		arc, _ := strconv.Atoi(m[1])
		episode, _ := strconv.Atoi(m[2])

		return &ParsedFilename{
			ArcNumber:  arc,
			EpisodeNum: episode,
			CRC32:      strings.ToUpper(m[3]),
			Extension:  m[4],
			Mode:       "library",
		}, nil
	}

	return nil, fmt.Errorf("invalid One Pace filename: %s", filename)
}

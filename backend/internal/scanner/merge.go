package scanner

import (
	"fmt"
	"path/filepath"

	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
)

// AddOrUpdateEpisode merges a scanned file + metadata into the library store.
// Each release version (e.g. "normal", "extended") is tracked independently,
// so importing one version never overwrites another already-imported version
// of the same episode.
func AddOrUpdateEpisode(
	lib *library.Library,
	filePath string,
	parsed *ParsedFilename,
	meta metadata.Episode,
	arcTitle string,
) library.Episode {

	// Determine real arc number
	arcNumber := parsed.ArcNumber
	if arcNumber == 0 {
		arcNumber = meta.Arc
	}

	arc := lib.GetOrCreateArc(arcNumber, arcTitle)
	key := fmt.Sprintf("%d", meta.Episode)

	existing, exists := arc.Episodes[key]
	if !exists {
		existing = library.Episode{EpisodeNumber: meta.Episode, Monitored: true}
	}
	if existing.Versions == nil {
		existing.Versions = map[string]library.EpisodeVersion{}
	}

	existing.Title = meta.Title
	existing.Description = meta.Description
	existing.Versions[meta.File.Version] = library.EpisodeVersion{
		CRC32:          meta.File.CRC32,
		FilePath:       filepath.ToSlash(filePath),
		DownloadStatus: "imported",
	}

	arc.Episodes[key] = existing
	return existing
}

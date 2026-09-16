package scanner

import (
	"fmt"
	"log"
	"os"
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

	newPath := filepath.ToSlash(filePath)
	if prev, ok := existing.Versions[meta.File.Version]; ok && prev.FilePath != "" && prev.FilePath != newPath {
		if err := os.Remove(prev.FilePath); err != nil && !os.IsNotExist(err) {
			log.Printf("merge: failed to remove superseded file %s: %v", prev.FilePath, err)
		}
	}

	existing.Title = meta.Title
	existing.Description = meta.Description
	existing.Versions[meta.File.Version] = library.EpisodeVersion{
		CRC32:          meta.File.CRC32,
		FilePath:       newPath,
		DownloadStatus: "imported",
	}

	arc.Episodes[key] = existing
	return existing
}

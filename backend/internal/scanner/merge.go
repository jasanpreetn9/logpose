package scanner

import (
	"fmt"
	"path/filepath"

	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
)

// AddOrUpdateEpisode merges a scanned file + metadata into the library store.
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

	entry := library.Episode{
		EpisodeNumber:  meta.Episode,
		CRC32:          meta.File.CRC32,
		Title:          meta.Title,
		Description:    meta.Description,
		Version:        meta.File.Version,
		FilePath:       filepath.ToSlash(filePath),
		DownloadStatus: "imported",
		Monitored:      existing.Monitored || !exists,
	}

	arc.Episodes[key] = entry
	return entry
}

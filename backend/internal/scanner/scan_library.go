package scanner

import (
	"log"
	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
	"onepace-library/internal/nfo"
	"os"
	"path/filepath"
	"strings"
)

type ScanStats struct {
	FilesFound         int
	FilesMarkedMissing int
	FilesImported      int
}

func ScanLibrary(root string, lib *library.Library, meta *metadata.Client) (ScanStats, error) {
	log.Printf("Starting library scan: %s", root)

	var stats ScanStats
	foundFiles := map[string]bool{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		filename := info.Name()
		if !(strings.HasSuffix(strings.ToLower(filename), ".mkv") ||
			strings.HasSuffix(strings.ToLower(filename), ".mp4")) {
			return nil
		}

		parsed, err := ParseOnePaceFilename(filename)
		if err != nil {
			log.Printf("Failed to parse filename %s: %v", filename, err)
			return nil
		}

		epMeta, err := meta.GetEpisodeByCRC32(parsed.CRC32)
		if err != nil {
			log.Printf("Metadata missing for CRC %s (%s)", parsed.CRC32, filename)
			return nil
		}

		arcTitle := meta.GetArcTitle(epMeta.Arc)
		entry := AddOrUpdateEpisode(lib, path, parsed, epMeta, arcTitle)
		foundFiles[path] = true
		stats.FilesFound++

		nfoPath := nfo.NFOPathForVideo(path)
		nfo.GenerateEpisodeNFO(entry, entry.Versions[epMeta.File.Version], epMeta, arcTitle, nfoPath)

		return nil
	})

	if err != nil {
		return stats, err
	}

	log.Println("Checking for removed files...")

	for arcNum, arc := range lib.Arcs {
		for epKey, ep := range arc.Episodes {
			changed := false
			for versionKey, v := range ep.Versions {
				if v.FilePath == "" {
					continue
				}
				if _, exists := foundFiles[v.FilePath]; !exists {
					log.Printf("File missing on disk, marking as missing: %s", v.FilePath)
					v.DownloadStatus = "missing"
					v.FilePath = ""
					ep.Versions[versionKey] = v
					stats.FilesMarkedMissing++
					changed = true
				}
			}
			if changed {
				lib.Arcs[arcNum].Episodes[epKey] = ep
			}
		}
	}

	log.Println("Library scan complete.")
	return stats, nil
}

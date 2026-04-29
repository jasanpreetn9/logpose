package scanner

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
	"onepace-library/internal/nfo"
)

type ImportedFile struct {
	Title   string
	SrcName string
	DstPath string
}

func ScanDownloads(downloadRoot, libraryRoot string, lib *library.Library, meta *metadata.Client) ([]ImportedFile, error) {

	log.Printf("Scanning downloads: %s\n", downloadRoot)

	var imported []ImportedFile

	err := filepath.Walk(downloadRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		name := info.Name()

		if !(strings.HasSuffix(strings.ToLower(name), ".mkv") ||
			strings.HasSuffix(strings.ToLower(name), ".mp4")) {
			return nil
		}

		parsed, err := ParseOnePaceFilename(name)
		if err != nil {
			log.Printf("Skipping non-OnePace file: %s", name)
			return nil
		}

		epMeta, err := meta.GetEpisodeByCRC32(parsed.CRC32)
		if err != nil {
			log.Printf("No metadata for CRC %s (file: %s)", parsed.CRC32, name)
			return nil
		}

		arcTitle := meta.GetArcTitle(epMeta.Arc)

		arcFolder := filepath.Join(
			libraryRoot,
			fmt.Sprintf("%02d - %s", epMeta.Arc, sanitizeFilename(arcTitle)),
		)
		os.MkdirAll(arcFolder, 0755)

		destFilename := fmt.Sprintf(
			"S%02dE%02d - %s [%s].%s",
			epMeta.Arc,
			epMeta.Episode,
			sanitizeFilename(epMeta.Title),
			parsed.CRC32,
			parsed.Extension,
		)

		dst := filepath.Join(arcFolder, destFilename)

		if err := moveFile(path, dst, libraryRoot); err != nil {
			log.Printf("Failed to move file: %v", err)
			return nil
		}

		entry := AddOrUpdateEpisode(lib, dst, parsed, epMeta, arcTitle)

		nfoPath := nfo.NFOPathForVideo(dst)
		nfo.GenerateEpisodeNFO(entry, epMeta, arcTitle, nfoPath)

		log.Printf("Imported: %s → %s", name, dst)
		imported = append(imported, ImportedFile{
			Title:   epMeta.Title,
			SrcName: name,
			DstPath: dst,
		})
		return nil
	})

	if err != nil {
		return imported, fmt.Errorf("scan downloads: %w", err)
	}

	log.Println("Downloads scan complete.")
	return imported, nil
}

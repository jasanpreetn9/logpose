// Package poller watches qBittorrent for completed torrents and imports them
// into the library automatically.
package poller

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"onepace-library/internal/activity"
	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
	"onepace-library/internal/nfo"
	"onepace-library/internal/qbittorrent"
	"onepace-library/internal/scanner"
)

// Start runs the completion poller until ctx is cancelled.
func Start(
	ctx context.Context,
	qb *qbittorrent.Client,
	meta *metadata.Client,
	store *library.Store,
	acts *activity.Store,
	libPath string,
) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := poll(qb, meta, store, acts, libPath); err != nil {
				log.Printf("poller: %v", err)
			}
		}
	}
}

func poll(
	qb *qbittorrent.Client,
	meta *metadata.Client,
	store *library.Store,
	acts *activity.Store,
	libPath string,
) error {
	torrents, err := qb.GetCompleted()
	if err != nil {
		return fmt.Errorf("GetCompleted: %w", err)
	}

	for _, t := range torrents {
		if err := importTorrent(t, qb, meta, store, acts, libPath); err != nil {
			log.Printf("poller: import %q: %v", t.Name, err)
		}
	}
	return nil
}

func importTorrent(
	t qbittorrent.TorrentInfo,
	qb *qbittorrent.Client,
	meta *metadata.Client,
	store *library.Store,
	acts *activity.Store,
	libPath string,
) error {
	contentPath := t.ContentPath
	if contentPath == "" {
		contentPath = filepath.Join(t.SavePath, t.Name)
	}

	// Single-file torrent: name parses directly as an episode filename.
	if parsed, err := scanner.ParseOnePaceFilename(t.Name); err == nil {
		return importFile(contentPath, parsed, qb, t.Hash, t.Name, meta, store, acts, libPath, true)
	}

	// Folder torrent: walk ContentPath for episode files.
	info, err := os.Stat(contentPath)
	if err != nil {
		return fmt.Errorf("stat content path %s: %w", contentPath, err)
	}
	if !info.IsDir() {
		// Not a directory and name didn't parse — nothing we can do.
		log.Printf("poller: skipping unrecognised torrent %q", t.Name)
		return nil
	}

	var importErr error
	anyImported := false

	err = filepath.Walk(contentPath, func(path string, fi os.FileInfo, werr error) error {
		if werr != nil || fi.IsDir() {
			return nil
		}
		lower := strings.ToLower(fi.Name())
		if !strings.HasSuffix(lower, ".mkv") && !strings.HasSuffix(lower, ".mp4") {
			return nil
		}
		parsed, err := scanner.ParseOnePaceFilename(fi.Name())
		if err != nil {
			log.Printf("poller: skipping %s (not an episode file)", fi.Name())
			return nil
		}
		if err := importFile(path, parsed, nil, "", fi.Name(), meta, store, acts, libPath, false); err != nil {
			log.Printf("poller: %s: %v", fi.Name(), err)
			importErr = err
		} else {
			anyImported = true
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk %s: %w", contentPath, err)
	}

	// Delete the torrent entry if at least one file imported successfully
	// and no hard errors remain, so we stop retrying forever.
	if anyImported && importErr == nil {
		log.Printf("poller: folder torrent %q fully imported, removing", t.Name)
		_ = qb.DeleteTorrent(t.Hash)
	} else if !anyImported && importErr == nil {
		// All files were skipped (unknown CRCs or non-episode files) — clean up.
		log.Printf("poller: folder torrent %q has no importable files, removing", t.Name)
		_ = qb.DeleteTorrent(t.Hash)
	}

	return importErr
}

// importFile imports a single episode file at srcPath. If deleteTorrent is true
// and qb/hash are provided, the torrent is removed from qBittorrent on success.
func importFile(
	srcPath string,
	parsed *scanner.ParsedFilename,
	qb *qbittorrent.Client,
	torrentHash string,
	logName string,
	meta *metadata.Client,
	store *library.Store,
	acts *activity.Store,
	libPath string,
	deleteTorrent bool,
) error {
	epMeta, err := meta.GetEpisodeByCRC32(parsed.CRC32)
	if err != nil {
		return fmt.Errorf("no metadata for CRC %s", parsed.CRC32)
	}

	arcTitle := meta.GetArcTitle(epMeta.Arc)
	arcFolder := filepath.Join(libPath,
		fmt.Sprintf("%02d - %s", epMeta.Arc, scanner.SanitizeFilename(arcTitle)))

	destFilename := fmt.Sprintf("S%02dE%02d - %s [%s].%s",
		epMeta.Arc, epMeta.Episode,
		scanner.SanitizeFilename(epMeta.Title),
		parsed.CRC32,
		parsed.Extension,
	)
	dst := filepath.Join(arcFolder, destFilename)

	// Already imported on a previous tick — just clean up the torrent entry.
	if _, err := os.Stat(dst); err == nil {
		log.Printf("poller: %s already imported, skipping", logName)
		if deleteTorrent && qb != nil {
			_ = qb.DeleteTorrent(torrentHash)
		}
		return nil
	}

	if err := os.MkdirAll(arcFolder, 0755); err != nil {
		return err
	}

	if err := scanner.MoveFile(srcPath, dst, libPath); err != nil {
		return fmt.Errorf("move %s → %s: %w", srcPath, dst, err)
	}

	var entry library.Episode
	if err := store.Write(func(lib *library.Library) error {
		entry = scanner.AddOrUpdateEpisode(lib, dst, parsed, epMeta, arcTitle)
		return nil
	}); err != nil {
		return fmt.Errorf("store write: %w", err)
	}

	nfoPath := nfo.NFOPathForVideo(dst)
	nfo.GenerateEpisodeNFO(entry, epMeta, arcTitle, nfoPath)

	acts.Add(activity.EventImport, "Imported: "+epMeta.Title, dst, true)
	log.Printf("poller: imported %s → %s", logName, dst)

	if deleteTorrent && qb != nil {
		_ = qb.DeleteTorrent(torrentHash)
	}
	return nil
}

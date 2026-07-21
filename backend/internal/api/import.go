package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"onepace-library/internal/activity"
	"onepace-library/internal/config"
	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
	"onepace-library/internal/nfo"
	"onepace-library/internal/scanner"
)

type UnmatchedFile struct {
	Path   string `json:"path"`
	Name   string `json:"name"`
	Reason string `json:"reason"` // "unparseable" or "unknown_crc"
	CRC32  string `json:"crc32,omitempty"`
}

// HandleGetUnmatched lists video files in the downloads folder that the
// scanner can't place: either the filename doesn't parse as a One Pace
// release, or it parses but the CRC isn't in metadata.
// GET /api/import/unmatched
func HandleGetUnmatched(meta *metadata.Client, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		files := []UnmatchedFile{}

		_ = filepath.Walk(cfg.DownloadPath, func(path string, fi os.FileInfo, werr error) error {
			if werr != nil || fi.IsDir() {
				return nil
			}
			lower := strings.ToLower(fi.Name())
			if !strings.HasSuffix(lower, ".mkv") && !strings.HasSuffix(lower, ".mp4") {
				return nil
			}

			parsed, err := scanner.ParseOnePaceFilename(fi.Name())
			if err != nil {
				files = append(files, UnmatchedFile{Path: path, Name: fi.Name(), Reason: "unparseable"})
				return nil
			}
			if _, err := meta.GetEpisodeByCRC32(parsed.CRC32); err != nil {
				files = append(files, UnmatchedFile{
					Path: path, Name: fi.Name(), Reason: "unknown_crc", CRC32: parsed.CRC32,
				})
			}
			return nil
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)
	}
}

// findEpisodeMeta finds the metadata for a specific (arc, episode, version).
// meta.EpisodesByArc collapses versions, so we scan the full CRC map instead.
func findEpisodeMeta(meta *metadata.Client, arc, episode int, version string) (metadata.Episode, bool) {
	for _, ep := range meta.Episodes() {
		if ep.Arc == arc && ep.Episode == episode && ep.File.Version == version {
			return ep, true
		}
	}
	return metadata.Episode{}, false
}

type manualImportRequest struct {
	Path    string `json:"path"`
	Arc     int    `json:"arc"`
	Episode int    `json:"episode"`
	Version string `json:"version"`
}

type manualImportPlan struct {
	epMeta   metadata.Episode
	arcTitle string
	crc32    string
	dst      string
}

func planManualImport(meta *metadata.Client, cfg *config.Config, req manualImportRequest) (*manualImportPlan, error) {
	if req.Version == "" {
		req.Version = "normal"
	}
	if _, err := os.Stat(req.Path); err != nil {
		return nil, fmt.Errorf("source file not found: %w", err)
	}

	epMeta, ok := findEpisodeMeta(meta, req.Arc, req.Episode, req.Version)
	if !ok {
		return nil, fmt.Errorf("no metadata for arc %d episode %d (%s)", req.Arc, req.Episode, req.Version)
	}

	crc, err := scanner.ComputeCRC32(req.Path)
	if err != nil {
		return nil, fmt.Errorf("compute CRC32: %w", err)
	}

	arcTitle := meta.GetArcTitle(req.Arc)
	arcFolder := filepath.Join(cfg.LibraryPath,
		fmt.Sprintf("%02d - %s", req.Arc, scanner.SanitizeFilename(arcTitle)))
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(req.Path)), ".")
	destFilename := fmt.Sprintf("S%02dE%02d - %s [%s].%s",
		req.Arc, req.Episode, scanner.SanitizeFilename(epMeta.Title), crc, ext)

	return &manualImportPlan{
		epMeta:   epMeta,
		arcTitle: arcTitle,
		crc32:    crc,
		dst:      filepath.Join(arcFolder, destFilename),
	}, nil
}

// HandleManualImportPreview resolves the metadata and destination filename
// for a chosen arc/episode/version without touching any files.
// GET /api/import/manual/preview?path=&arc=&episode=&version=
func HandleManualImportPreview(meta *metadata.Client, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		arc, _ := strconv.Atoi(q.Get("arc"))
		episode, _ := strconv.Atoi(q.Get("episode"))
		req := manualImportRequest{
			Path:    q.Get("path"),
			Arc:     arc,
			Episode: episode,
			Version: q.Get("version"),
		}

		plan, err := planManualImport(meta, cfg, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"title":        plan.epMeta.Title,
			"arc":          plan.epMeta.Arc,
			"episode":      plan.epMeta.Episode,
			"version":      plan.epMeta.File.Version,
			"destFolder":   plan.arcTitle,
			"destFilename": filepath.Base(plan.dst),
		})
	}
}

// HandleManualImport imports a file the scanner couldn't match, using the
// arc/episode/version the user picked. Repeats the same lookup as the
// preview endpoint, then actually moves the file into the library.
// POST /api/import/manual
func HandleManualImport(meta *metadata.Client, cfg *config.Config, store *library.Store, acts *activity.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req manualImportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		plan, err := planManualImport(meta, cfg, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if _, err := os.Stat(plan.dst); err == nil {
			http.Error(w, "destination file already exists: "+plan.dst, http.StatusConflict)
			return
		}
		if err := os.MkdirAll(filepath.Dir(plan.dst), 0755); err != nil {
			http.Error(w, "create arc folder: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := scanner.MoveFile(req.Path, plan.dst, cfg.LibraryPath); err != nil {
			http.Error(w, "move file: "+err.Error(), http.StatusInternalServerError)
			return
		}

		epMeta := plan.epMeta
		epMeta.File.CRC32 = plan.crc32 // ground truth, not the metadata's own CRC

		parsed := &scanner.ParsedFilename{
			ArcNumber:  plan.epMeta.Arc,
			EpisodeNum: plan.epMeta.Episode,
			CRC32:      plan.crc32,
			Extension:  strings.TrimPrefix(filepath.Ext(plan.dst), "."),
		}

		var entry library.Episode
		if err := store.Write(func(lib *library.Library) error {
			entry = scanner.AddOrUpdateEpisode(lib, plan.dst, parsed, epMeta, plan.arcTitle)
			return nil
		}); err != nil {
			http.Error(w, "store write: "+err.Error(), http.StatusInternalServerError)
			return
		}

		nfoPath := nfo.NFOPathForVideo(plan.dst)
		nfo.GenerateEpisodeNFO(entry, epMeta, plan.arcTitle, nfoPath)

		acts.Add(activity.EventImport, "Imported: "+epMeta.Title, plan.dst, true)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"title":  epMeta.Title,
			"path":   plan.dst,
		})
	}
}

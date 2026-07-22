package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"onepace-library/internal/activity"
	"onepace-library/internal/config"
	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
	"onepace-library/internal/scanner"
)

type renameOp struct {
	CRC     string `json:"-"`
	Arc     int    `json:"-"`
	Episode int    `json:"-"`
	OldPath string `json:"-"`
	NewPath string `json:"-"`

	Folder string `json:"folder"`
	From   string `json:"from"`
	To     string `json:"to"`
}

// planRenames walks the library and returns the renames needed to bring every
// recognizable episode file to the canonical "S##E## - Title [CRC].ext" name.
// total counts all recognizable video files, renamed or not.
func planRenames(meta *metadata.Client, libraryPath string) (ops []renameOp, total int, err error) {
	err = filepath.Walk(libraryPath, func(path string, fi os.FileInfo, werr error) error {
		if werr != nil || fi.IsDir() {
			return nil
		}
		base := fi.Name()
		lower := strings.ToLower(base)
		if !strings.HasSuffix(lower, ".mkv") && !strings.HasSuffix(lower, ".mp4") {
			return nil
		}
		// Never touch files still staged in .tmp.
		if strings.Contains(path, string(filepath.Separator)+".tmp"+string(filepath.Separator)) {
			return nil
		}

		parsed, err := scanner.ParseOnePaceFilename(base)
		if err != nil {
			return nil
		}
		total++

		epMeta, err := meta.GetEpisodeByCRC32(parsed.CRC32)
		if err != nil {
			log.Printf("rename: no metadata for CRC %s (%s), skipping", parsed.CRC32, base)
			return nil
		}

		canonical := fmt.Sprintf("S%02dE%02d - %s [%s].%s",
			epMeta.Arc, epMeta.Episode,
			scanner.SanitizeFilename(epMeta.Title),
			epMeta.File.CRC32,
			parsed.Extension,
		)
		if base == canonical {
			return nil
		}

		dir := filepath.Dir(path)
		newPath := filepath.Join(dir, canonical)
		if _, err := os.Stat(newPath); err == nil {
			log.Printf("rename: %s already exists, skipping %s", canonical, base)
			return nil
		}

		folder, relErr := filepath.Rel(libraryPath, dir)
		if relErr != nil {
			folder = dir
		}

		ops = append(ops, renameOp{
			CRC:     epMeta.File.CRC32,
			Arc:     epMeta.Arc,
			Episode: epMeta.Episode,
			OldPath: path,
			NewPath: newPath,
			Folder:  folder,
			From:    base,
			To:      canonical,
		})
		return nil
	})
	return ops, total, err
}

// HandleRenamePreview returns the renames that would be performed, without
// touching any files.
// GET /api/library/rename/preview
func HandleRenamePreview(meta *metadata.Client, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ops, total, err := planRenames(meta, cfg.LibraryPath)
		if err != nil {
			http.Error(w, "walk library: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if ops == nil {
			ops = []renameOp{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"total":   total,
			"renames": ops,
		})
	}
}

// HandleRenameFiles renames every recognizable episode file in the library to
// the canonical "S##E## - Title [CRC].ext" format, along with its .nfo and
// -thumb.jpg sidecars. Files whose CRC is unknown to the metadata are left
// untouched.
// POST /api/library/rename
func HandleRenameFiles(meta *metadata.Client, cfg *config.Config, store *library.Store, acts *activity.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ops, total, err := planRenames(meta, cfg.LibraryPath)
		if err != nil {
			http.Error(w, "walk library: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var done []renameOp
		for _, op := range ops {
			if err := os.Rename(op.OldPath, op.NewPath); err != nil {
				log.Printf("rename: %s: %v", op.From, err)
				continue
			}

			// Move matching sidecars along with the video.
			oldStem := strings.TrimSuffix(op.OldPath, filepath.Ext(op.OldPath))
			newStem := strings.TrimSuffix(op.NewPath, filepath.Ext(op.NewPath))
			for _, suffix := range []string{".nfo", "-thumb.jpg"} {
				if _, err := os.Stat(oldStem + suffix); err == nil {
					_ = os.Rename(oldStem+suffix, newStem+suffix)
				}
			}

			log.Printf("rename: %s → %s", op.From, op.To)
			done = append(done, op)
		}

		// Point tracked episodes at their new paths (matched by CRC, since
		// stored paths may use a different mount prefix than this process).
		if len(done) > 0 {
			if err := store.Write(func(lib *library.Library) error {
				for _, op := range done {
					arc, ok := lib.Arcs[op.Arc]
					if !ok {
						continue
					}
					key := fmt.Sprintf("%d", op.Episode)
					ep, ok := arc.Episodes[key]
					if !ok {
						continue
					}
					for versionKey, v := range ep.Versions {
						if !strings.EqualFold(v.CRC32, op.CRC) {
							continue
						}
						v.FilePath = op.NewPath
						ep.Versions[versionKey] = v
						arc.Episodes[key] = ep
						break
					}
				}
				return nil
			}); err != nil {
				log.Printf("rename: store update failed: %v", err)
			}
		}

		acts.Add(activity.EventLibraryScan,
			fmt.Sprintf("Rename complete — %d/%d files renamed", len(done), total),
			"",
			true,
		)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"renamed": len(done),
			"total":   total,
		})
	}
}

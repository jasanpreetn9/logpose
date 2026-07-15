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

// HandleRenameFiles renames every recognizable episode file in the library to
// the canonical "S##E## - Title [CRC].ext" format, along with its .nfo and
// -thumb.jpg sidecars. Files whose CRC is unknown to the metadata are left
// untouched.
// POST /api/library/rename
func HandleRenameFiles(meta *metadata.Client, cfg *config.Config, store *library.Store, acts *activity.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		type renameOp struct {
			crc     string
			arc     int
			episode int
			newPath string
		}
		var ops []renameOp

		total := 0
		renamed := 0

		err := filepath.Walk(cfg.LibraryPath, func(path string, fi os.FileInfo, werr error) error {
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
			if err := os.Rename(path, newPath); err != nil {
				log.Printf("rename: %s: %v", base, err)
				return nil
			}

			// Move matching sidecars along with the video.
			oldStem := strings.TrimSuffix(path, filepath.Ext(path))
			newStem := strings.TrimSuffix(newPath, filepath.Ext(newPath))
			for _, suffix := range []string{".nfo", "-thumb.jpg"} {
				if _, err := os.Stat(oldStem + suffix); err == nil {
					_ = os.Rename(oldStem+suffix, newStem+suffix)
				}
			}

			log.Printf("rename: %s → %s", base, canonical)
			ops = append(ops, renameOp{
				crc:     epMeta.File.CRC32,
				arc:     epMeta.Arc,
				episode: epMeta.Episode,
				newPath: newPath,
			})
			renamed++
			return nil
		})
		if err != nil {
			http.Error(w, "walk library: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Point tracked episodes at their new paths (matched by CRC, since
		// stored paths may use a different mount prefix than this process).
		if len(ops) > 0 {
			if err := store.Write(func(lib *library.Library) error {
				for _, op := range ops {
					arc, ok := lib.Arcs[op.arc]
					if !ok {
						continue
					}
					key := fmt.Sprintf("%d", op.episode)
					ep, ok := arc.Episodes[key]
					if !ok || !strings.EqualFold(ep.CRC32, op.crc) {
						continue
					}
					ep.FilePath = op.newPath
					arc.Episodes[key] = ep
				}
				return nil
			}); err != nil {
				log.Printf("rename: store update failed: %v", err)
			}
		}

		acts.Add(activity.EventLibraryScan,
			fmt.Sprintf("Rename complete — %d/%d files renamed", renamed, total),
			"",
			true,
		)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"renamed": renamed,
			"total":   total,
		})
	}
}

package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"onepace-library/internal/library"
	"onepace-library/internal/metadata"

	"github.com/go-chi/chi/v5"
)

func HandleGetEpisode(meta *metadata.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		crc := chi.URLParam(r, "crc")

		ep, err := meta.GetEpisodeByCRC32(crc)
		if err != nil {
			http.Error(w, "episode not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ep)
	}
}

// removeEpisodeFiles deletes a version's video file and its .nfo/-thumb.jpg
// sidecars. The library record (not disk state) is the source of truth for
// what's "imported", so a file already missing on disk is not an error.
func removeEpisodeFiles(filePath string) {
	if filePath == "" {
		return
	}
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		log.Printf("delete: failed to remove %s: %v", filePath, err)
	}
	stem := strings.TrimSuffix(filePath, filepath.Ext(filePath))
	for _, suffix := range []string{".nfo", "-thumb.jpg"} {
		sidecar := stem + suffix
		if err := os.Remove(sidecar); err != nil && !os.IsNotExist(err) {
			log.Printf("delete: failed to remove %s: %v", sidecar, err)
		}
	}
}

// HandleDeleteEpisodeVersion deletes one downloaded version's file and
// library record, reverting that version back to "missing".
// DELETE /api/episodes/{crc}
func HandleDeleteEpisodeVersion(store *library.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		crc := chi.URLParam(r, "crc")
		if crc == "" {
			http.Error(w, "missing crc", http.StatusBadRequest)
			return
		}

		var filePath string
		found := false

		if err := store.Write(func(lib *library.Library) error {
			for arcNum, arc := range lib.Arcs {
				for epKey, ep := range arc.Episodes {
					for versionKey, v := range ep.Versions {
						if v.CRC32 != crc {
							continue
						}
						found = true
						filePath = v.FilePath
						delete(ep.Versions, versionKey)
						lib.Arcs[arcNum].Episodes[epKey] = ep
						return nil
					}
				}
			}
			return nil
		}); err != nil {
			http.Error(w, "failed to update library", http.StatusInternalServerError)
			return
		}

		if !found {
			http.Error(w, "version not found in library", http.StatusNotFound)
			return
		}

		removeEpisodeFiles(filePath)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	}
}

// HandleDeleteEpisode deletes every downloaded version's file for one
// episode, reverting the whole episode back to "missing".
// DELETE /api/episodes/{arc}/{episode}
func HandleDeleteEpisode(store *library.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		arcNum, err := strconv.Atoi(chi.URLParam(r, "arc"))
		if err != nil {
			http.Error(w, "invalid arc", http.StatusBadRequest)
			return
		}
		epNum, err := strconv.Atoi(chi.URLParam(r, "episode"))
		if err != nil {
			http.Error(w, "invalid episode", http.StatusBadRequest)
			return
		}
		key := strconv.Itoa(epNum)

		var filePaths []string

		if err := store.Write(func(lib *library.Library) error {
			arc, ok := lib.Arcs[arcNum]
			if !ok {
				return nil
			}
			ep, ok := arc.Episodes[key]
			if !ok {
				return nil
			}
			for versionKey, v := range ep.Versions {
				if v.FilePath != "" {
					filePaths = append(filePaths, v.FilePath)
				}
				delete(ep.Versions, versionKey)
			}
			arc.Episodes[key] = ep
			return nil
		}); err != nil {
			http.Error(w, "failed to update library", http.StatusInternalServerError)
			return
		}

		for _, p := range filePaths {
			removeEpisodeFiles(p)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "deleted", "count": len(filePaths)})
	}
}

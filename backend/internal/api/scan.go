package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"onepace-library/internal/activity"
	"onepace-library/internal/config"
	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
	"onepace-library/internal/scanner"
)

func HandleScanLibrary(meta *metadata.Client, cfg *config.Config, store *library.Store, acts *activity.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var stats scanner.ScanStats
		err := store.TryScan(func(lib *library.Library) error {
			var e error
			stats, e = scanner.ScanLibrary(cfg.LibraryPath, lib, meta)
			return e
		})
		if err != nil {
			acts.Add(activity.EventLibraryScan, "Library scan failed", err.Error(), false)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		acts.Add(
			activity.EventLibraryScan,
			"Library scan complete",
			fmt.Sprintf("%d files found, %d marked missing", stats.FilesFound, stats.FilesMarkedMissing),
			true,
		)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":             "ok",
			"message":            "library scan complete",
			"filesFound":         stats.FilesFound,
			"filesMarkedMissing": stats.FilesMarkedMissing,
		})
	}
}

func HandleScanDownloads(meta *metadata.Client, cfg *config.Config, store *library.Store, acts *activity.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var imported []scanner.ImportedFile
		err := store.TryScan(func(lib *library.Library) error {
			var e error
			imported, e = scanner.ScanDownloads(cfg.DownloadPath, cfg.LibraryPath, lib, meta)
			return e
		})
		if err != nil {
			acts.Add(activity.EventDownloadsScan, "Downloads scan failed", err.Error(), false)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		acts.Add(
			activity.EventDownloadsScan,
			fmt.Sprintf("Downloads scan complete — %d file(s) imported", len(imported)),
			"",
			true,
		)
		for _, f := range imported {
			acts.Add(activity.EventImport, "Imported: "+f.Title, f.DstPath, true)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"message": "download scan complete",
			"imported": len(imported),
		})
	}
}

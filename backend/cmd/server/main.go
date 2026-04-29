package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"onepace-library/internal/activity"
	"onepace-library/internal/api"
	"onepace-library/internal/config"
	"onepace-library/internal/db"
	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
	"onepace-library/internal/nfo"
	"onepace-library/internal/poller"
	"onepace-library/internal/qbittorrent"
	"onepace-library/internal/scanner"
	"onepace-library/internal/sse"
	"onepace-library/internal/watcher"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	cfgPath := envOr("CONFIG_PATH", "../config.yml")
	dataDir := envOr("DATA_DIR", "data")

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	database, err := db.Open(dataDir + "/library.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	metaClient := metadata.NewClient(cfg.Metadata.EpisodesURL, cfg.Metadata.ArcsURL)
	if err := metaClient.Refresh(); err != nil {
		log.Fatalf("Failed to load metadata: %v", err)
	}

	hub := sse.NewHub()
	acts := activity.NewStore(database, hub)
	acts.LoadFromDB(500)

	store := library.NewStore(cfg.LibraryJSONPath, database)

	qb := qbittorrent.NewClient(
		cfg.QBittorrent.Host,
		cfg.QBittorrent.Username,
		cfg.QBittorrent.Password,
	)
	if cfg.QBittorrent.Enabled {
		if err := qb.Login(); err != nil {
			log.Printf("WARNING: qBittorrent not reachable at startup (%s): %v", cfg.QBittorrent.Host, err)
		} else {
			log.Printf("qBittorrent connected: %s", cfg.QBittorrent.Host)
		}
	}

	tickerReset := make(chan time.Duration, 1)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(api.CORS)

	api.RegisterRoutes(r, metaClient, cfg, cfgPath, store, qb, acts, hub, tickerReset)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Configurable metadata refresh ticker.
	go func() {
		ticker := time.NewTicker(cfg.RefreshDuration())
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case newDur := <-tickerReset:
				ticker.Stop()
				ticker = time.NewTicker(newDur)
				log.Printf("Metadata refresh interval updated to %s", newDur)
			case <-ticker.C:
				if err := metaClient.Refresh(); err != nil {
					log.Printf("Metadata refresh failed: %v", err)
					continue
				}
				log.Println("Metadata refreshed.")
				regenStaleNFOs(metaClient, store)
			}
		}
	}()

	// qBittorrent completion poller.
	if cfg.QBittorrent.Enabled {
		go poller.Start(ctx, qb, metaClient, store, acts, cfg.LibraryPath)
	}

	// fsnotify watcher on downloads directory.
	go func() {
		triggerScan := func() {
			var imported []scanner.ImportedFile
			err := store.TryScan(func(lib *library.Library) error {
				var e error
				imported, e = scanner.ScanDownloads(cfg.DownloadPath, cfg.LibraryPath, lib, metaClient)
				return e
			})
			if err != nil {
				acts.Add(activity.EventDownloadsScan, "Downloads scan failed", err.Error(), false)
				return
			}
			acts.Add(activity.EventDownloadsScan,
				"Downloads scan complete (auto)", "", true)
			for _, f := range imported {
				acts.Add(activity.EventImport, "Imported: "+f.Title, f.DstPath, true)
			}
		}
		if err := watcher.Start(ctx, cfg.DownloadPath, triggerScan); err != nil {
			log.Printf("watcher: %v", err)
		}
	}()

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("Server started on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	cancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	if err := server.Shutdown(shutCtx); err != nil {
		log.Fatalf("Shutdown failed: %v", err)
	}
	log.Println("Server exited gracefully.")
}

// regenStaleNFOs re-generates NFO files for any episodes whose metadata changed.
func regenStaleNFOs(meta *metadata.Client, store *library.Store) {
	stale := meta.StaleEpisodes()
	if len(stale) == 0 {
		return
	}
	store.Read(func(lib *library.Library) {
		for _, arc := range lib.Arcs {
			for _, ep := range arc.Episodes {
				for _, crc := range stale {
					if ep.CRC32 == crc && ep.FilePath != "" {
						epMeta, err := meta.GetEpisodeByCRC32(crc)
						if err != nil {
							continue
						}
						arcTitle := meta.GetArcTitle(arc.ArcNumber)
						nfoPath := nfo.NFOPathForVideo(ep.FilePath)
						nfo.GenerateEpisodeNFO(ep, epMeta, arcTitle, nfoPath)
						log.Printf("Regenerated NFO for %s (CRC %s)", ep.Title, crc)
					}
				}
			}
		}
	})
}

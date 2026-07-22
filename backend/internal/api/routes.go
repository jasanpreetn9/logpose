package api

import (
	"time"

	"github.com/go-chi/chi/v5"

	"onepace-library/internal/activity"
	"onepace-library/internal/config"
	"onepace-library/internal/downloads"
	"onepace-library/internal/library"
	"onepace-library/internal/metadata"
	"onepace-library/internal/qbittorrent"
	"onepace-library/internal/sse"
)

func RegisterRoutes(
	r chi.Router,
	meta *metadata.Client,
	cfg *config.Config,
	cfgPath string,
	store *library.Store,
	qb *qbittorrent.Client,
	acts *activity.Store,
	hub *sse.Hub,
	tracker *downloads.Tracker,
	tickerReset chan<- time.Duration,
) {
	r.Route("/api", func(api chi.Router) {

		api.Get("/library", HandleGetLibrary(store))

		api.Post("/scan/library", HandleScanLibrary(meta, cfg, store, acts))
		api.Post("/scan/downloads", HandleScanDownloads(meta, cfg, store, acts))
		api.Get("/library/rename/preview", HandleRenamePreview(meta, cfg))
		api.Post("/library/rename", HandleRenameFiles(meta, cfg, store, acts))

		api.Get("/episodes/all", HandleGetAllEpisodes(meta, store, tracker))
		api.Get("/episodes/{crc}", HandleGetEpisode(meta))
		api.Post("/episodes/monitor", HandleMonitorEpisode(meta, store))

		api.Post("/download/add", HandleAddToQbit(meta, qb, acts, tracker, cfg.QBittorrent.Enabled))

		api.Post("/arcs/{arcId}/monitor", HandleMonitorArc(meta, store))
		api.Post("/arcs/{arcId}/download-monitored", HandleDownloadMonitored(meta, store, qb, acts, tracker, cfg.QBittorrent.Enabled))
		api.Post("/arcs/{arcId}/verify-nfo", HandleVerifyNFOs(meta, store, acts))

		api.Get("/queue", HandleGetQueue(meta, qb, tracker, cfg.QBittorrent.Enabled))
		api.Delete("/queue/{hash}", HandleDeleteQueueItem(qb, cfg.QBittorrent.Enabled))

		api.Get("/import/unmatched", HandleGetUnmatched(meta, cfg))
		api.Get("/import/manual/preview", HandleManualImportPreview(meta, cfg))
		api.Post("/import/manual", HandleManualImport(meta, cfg, store, acts))

		api.Get("/activity", HandleGetActivity(acts))
		api.Get("/history", HandleGetHistory(acts))

		api.Post("/metadata/refresh", HandleRefreshMetadata(meta, cfg, store, acts, qb, tracker))

		api.Post("/qbittorrent/test", HandleTestQBittorrent(cfg))

		api.Get("/version", HandleGetVersion())
		api.Get("/health", HandleHealth(cfg, meta, qb))

		api.Get("/config", HandleGetConfig(cfg))
		api.Post("/config", HandleUpdateConfig(cfg, cfgPath, qb, tickerReset))

		api.Get("/events", sse.HandleSSE(hub))
	})
}

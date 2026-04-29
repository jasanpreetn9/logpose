package api

import (
	"time"

	"github.com/go-chi/chi/v5"

	"onepace-library/internal/activity"
	"onepace-library/internal/config"
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
	tickerReset chan<- time.Duration,
) {
	r.Route("/api", func(api chi.Router) {

		api.Get("/library", HandleGetLibrary(store))

		api.Post("/scan/library", HandleScanLibrary(meta, cfg, store, acts))
		api.Post("/scan/downloads", HandleScanDownloads(meta, cfg, store, acts))

		api.Get("/episodes/all", HandleGetAllEpisodes(meta, store))
		api.Get("/episodes/{crc}", HandleGetEpisode(meta))
		api.Post("/episodes/monitor", HandleMonitorEpisode(meta, store))

		api.Post("/download/add", HandleAddToQbit(meta, qb, acts, cfg.QBittorrent.Enabled))

		api.Post("/arcs/{arcId}/monitor", HandleMonitorArc(meta, store))
		api.Post("/arcs/{arcId}/download-monitored", HandleDownloadMonitored(meta, store, qb, acts, cfg.QBittorrent.Enabled))
		api.Post("/arcs/{arcId}/verify-nfo", HandleVerifyNFOs(meta, store, acts))

		api.Get("/activity", HandleGetActivity(acts))
		api.Get("/history", HandleGetHistory(acts))

		api.Get("/config", HandleGetConfig(cfg))
		api.Post("/config", HandleUpdateConfig(cfg, cfgPath, tickerReset))

		api.Get("/events", sse.HandleSSE(hub))
	})
}

// lib/types.ts
type UnifiedEpisode = {
    arc: number;
    episode: number;
    title: string;
    description: string;
    released: string;
    downloaded: boolean;
    monitored: boolean;
    versions: EpisodeVersion[];
}

type EpisodeVersion = {
    crc32: string;
    version: "normal" | "extended";
    file_path: string | null;
    status: "imported" | "missing" | "upgradable";
}

type UnifiedArc = {
    arc: number;
    title: string;
    audioLanguages: string;
    subtitleLanguages: string;
    resolution: string;
    status: string;

    mangaChapters: string | null;
    numberOfChapters: string | null;
    animeEpisodes: string | null;
    episodesAdapted: string | null;
    fillerEpisodes: string | null;
    timeSavedMins: string | null;
    timeSavedPercent: string | null;

    episodeCount: number;
    episodesDownloaded: number;
    episodes: UnifiedEpisode[];
}

type ActivityEvent = {
    id: string;
    type: 'download_queued' | 'download_failed' | 'library_scan' | 'downloads_scan' | 'import';
    timestamp: string;
    message: string;
    details: string;
    success: boolean;
}

type AppConfig = {
    port: string;
    libraryPath: string;
    downloadPath: string;
    libraryJsonPath: string;
    metadataEpisodesUrl: string;
    metadataArcsUrl: string;
    metadataRefreshInterval: string;
    qbEnabled: boolean;
    qbHost: string;
    qbUsername: string;
}

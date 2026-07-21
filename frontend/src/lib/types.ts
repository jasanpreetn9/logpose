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
    status: "imported" | "missing" | "upgradable" | "queued";
}

type RenamePreviewItem = {
    folder: string;
    from: string;
    to: string;
}

type QueueItem = {
    hash: string;
    name: string;
    title: string;
    arc: number;
    episode: number;
    progress: number;
    size: number;
    dlspeed: number;
    eta: number;
    state: string;
}

type UnmatchedFile = {
    path: string;
    name: string;
    reason: 'unparseable' | 'unknown_crc';
    crc32?: string;
}

type ManualImportPreview = {
    title: string;
    arc: number;
    episode: number;
    version: string;
    destFolder: string;
    destFilename: string;
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

type HealthCheck = {
    id: string;
    level: 'warning' | 'error';
    message: string;
}

type ActivityEvent = {
    id: string;
    type: 'download_queued' | 'download_failed' | 'library_scan' | 'downloads_scan' | 'import' | 'metadata_refresh';
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
    autoDownload: boolean;
    discordWebhookUrl: string;
    jellyfinUrl: string;
}

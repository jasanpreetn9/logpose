const BASE_URL = '/api';

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const res = await fetch(`${BASE_URL}${endpoint}`, {
        headers: {
            'Content-Type': 'application/json',
            ...(options.headers ?? {})
        },
        ...options
    });

    if (!res.ok) {
        const text = await res.text();
        console.error(`API Error ${res.status}:`, text);
        throw new Error(text || `API error: ${res.status}`);
    }

    return res.json() as Promise<T>;
}

/* -------------------------------------------------------
   Raw API shapes (matches Go JSON output)
------------------------------------------------------- */

type RawVersion = {
    crc32: string;
    version: string;
    file_path: string;
    status: string;
};

type RawEpisode = {
    arc: number;
    episode: number;
    title: string;
    description: string;
    released: string;
    downloaded: boolean;
    monitored: boolean;
    versions: RawVersion[];
};

type RawArc = {
    arc: number;
    title: string;
    audio_languages: string;
    subtitle_languages: string;
    resolution: string;
    status: string;
    manga_chapters: string;
    number_of_chapters: string;
    anime_episodes: string;
    episodes_adapted: string;
    filler_episodes: string;
    time_saved_mins: string;
    time_saved_percent: string;
    episode_count: number;
    episode_downloaded: number;
    episodes: RawEpisode[];
};

/* -------------------------------------------------------
   Mapping Helpers
------------------------------------------------------- */

function mapVersion(v: RawVersion): EpisodeVersion {
    return {
        crc32: v.crc32,
        version: v.version as EpisodeVersion['version'],
        file_path: v.file_path || null,
        status: v.status as EpisodeVersion['status'],
    };
}

function mapEpisode(ep: RawEpisode): UnifiedEpisode {
    return {
        arc: ep.arc,
        episode: ep.episode,
        title: ep.title,
        description: ep.description,
        released: ep.released,
        downloaded: ep.downloaded,
        monitored: ep.monitored,
        versions: ep.versions.map(mapVersion),
    };
}

function mapArc(a: RawArc): UnifiedArc {
    return {
        arc: a.arc,
        title: a.title,
        audioLanguages: a.audio_languages,
        subtitleLanguages: a.subtitle_languages,
        resolution: a.resolution,
        status: a.status,

        mangaChapters: a.manga_chapters || null,
        numberOfChapters: a.number_of_chapters || null,
        animeEpisodes: a.anime_episodes || null,
        episodesAdapted: a.episodes_adapted || null,
        fillerEpisodes: a.filler_episodes || null,
        timeSavedMins: a.time_saved_mins || null,
        timeSavedPercent: a.time_saved_percent || null,

        episodeCount: a.episode_count,
        episodesDownloaded: a.episode_downloaded,

        episodes: a.episodes.map(mapEpisode)
    };
}

/* -------------------------------------------------------
   Public API
------------------------------------------------------- */

export const api = {
    async getAllEpisodes(): Promise<UnifiedArc[]> {
        const raw = await request<RawArc[]>('/episodes/all');
        return raw.map(mapArc);
    },

    async scanLibrary(): Promise<void> {
        await request('/scan/library', { method: 'POST' });
    },

    async scanDownloads(): Promise<void> {
        await request('/scan/downloads', { method: 'POST' });
    },

    async toggleMonitor(arc: number, episode: number, monitored: boolean): Promise<void> {
        await request('/episodes/monitor', {
            method: 'POST',
            body: JSON.stringify({ arc, episode, monitored })
        });
    },

    async downloadEpisode(crc32: string): Promise<void> {
        await request('/download/add', {
            method: 'POST',
            body: JSON.stringify({ crc32 })
        });
    },

    async getActivity(): Promise<ActivityEvent[]> {
        return request<ActivityEvent[]>('/activity');
    },

    async getHistory(): Promise<ActivityEvent[]> {
        return request<ActivityEvent[]>('/history');
    },

    async monitorArc(arcId: number, monitored: boolean): Promise<void> {
        await request(`/arcs/${arcId}/monitor`, {
            method: 'POST',
            body: JSON.stringify({ monitored })
        });
    },

    async downloadMonitored(arcId: number): Promise<{ queued: number; total: number }> {
        return request(`/arcs/${arcId}/download-monitored`, { method: 'POST' });
    },

    async verifyNFOs(arcId: number): Promise<{ updated: number; total: number }> {
        return request(`/arcs/${arcId}/verify-nfo`, { method: 'POST' });
    },

    async getConfig(): Promise<AppConfig> {
        return request<AppConfig>('/config');
    },

    async updateConfig(patch: Partial<AppConfig> & { qbPassword?: string }): Promise<{ errors?: Record<string, string> }> {
        const res = await fetch(`${BASE_URL}/config`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(patch)
        });
        const body = await res.json();
        if (!res.ok) {
            return body as { errors: Record<string, string> };
        }
        return {};
    }
};

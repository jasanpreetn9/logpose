// Central color/label lookup tables for status pills across the app.
// Mirrors the mockup's statusMeta / stateMeta / typeMeta / qbitColors tables.

export type EpisodeStatus = 'imported' | 'importing' | 'queued' | 'upgradable' | 'missing';

export const episodeStatusMeta: Record<EpisodeStatus, { color: string; bg: string; label: string }> = {
	imported: { color: '#3ecf8e', bg: '#122019', label: 'IMPORTED' },
	importing: { color: '#f5a623', bg: '#2a2213', label: 'IMPORTING' },
	queued: { color: '#4d9fff', bg: '#152233', label: 'QUEUED' },
	upgradable: { color: '#b18cf5', bg: '#25203a', label: 'UPGRADE' },
	missing: { color: '#6b7280', bg: '#20242b', label: 'MISSING' }
};

export const queueStateMeta: Record<string, { color: string; bg: string; label: string }> = {
	downloading: { color: '#f5a623', bg: '#2a2213', label: 'Downloading' },
	importing: { color: '#f5a623', bg: '#2a2213', label: 'Importing' },
	stalledDL: { color: '#6b7280', bg: '#20242b', label: 'Stalled' },
	metaDL: { color: '#6b7280', bg: '#20242b', label: 'Fetching metadata' },
	queuedDL: { color: '#6b7280', bg: '#20242b', label: 'Queued' },
	pausedDL: { color: '#6b7280', bg: '#20242b', label: 'Paused' },
	forcedDL: { color: '#f5a623', bg: '#2a2213', label: 'Downloading' },
	checkingDL: { color: '#6b7280', bg: '#20242b', label: 'Checking' },
	uploading: { color: '#3ecf8e', bg: '#122019', label: 'Seeding' },
	stalledUP: { color: '#3ecf8e', bg: '#122019', label: 'Seeding' },
	checkingUP: { color: '#6b7280', bg: '#20242b', label: 'Checking' },
	error: { color: '#e5484d', bg: '#2a1416', label: 'Error' },
	missingFiles: { color: '#e5484d', bg: '#2a1416', label: 'Missing files' },
	allocating: { color: '#6b7280', bg: '#20242b', label: 'Allocating' }
};

export function queueStateStyle(state: string): { color: string; bg: string; label: string } {
	return queueStateMeta[state] ?? { color: '#6b7280', bg: '#20242b', label: state };
}

export const activityTypeMeta: Record<string, { color: string; label: string }> = {
	download_queued: { color: '#4d9fff', label: 'QUEUED' },
	download_failed: { color: '#e5484d', label: 'FAILED' },
	library_scan: { color: '#8992a0', label: 'LIB SCAN' },
	downloads_scan: { color: '#8992a0', label: 'DL SCAN' },
	import: { color: '#3ecf8e', label: 'IMPORT' },
	metadata_refresh: { color: '#b18cf5', label: 'METADATA' }
};

export function activityTypeStyle(type: string): { color: string; label: string } {
	return activityTypeMeta[type] ?? { color: '#8992a0', label: type };
}

export type QbitStatus = 'connected' | 'testing' | 'error' | 'disabled';

export const qbitColors: Record<QbitStatus, string> = {
	connected: '#3ecf8e',
	testing: '#f5a623',
	error: '#e5484d',
	disabled: '#6b7280'
};

/** Bytes → human-readable size, e.g. 1552428800 -> "1.4 GiB". */
export function fmtBytes(bytes: number): string {
	if (!bytes || bytes <= 0) return '0 B';
	const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
	let i = 0;
	let n = bytes;
	while (n >= 1024 && i < units.length - 1) {
		n /= 1024;
		i++;
	}
	return `${n.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

/** Bytes/sec → human-readable speed, e.g. "2.0 MiB/s". 0 means stalled. */
export function fmtSpeed(bytesPerSec: number): string {
	if (!bytesPerSec || bytesPerSec <= 0) return '—';
	return `${fmtBytes(bytesPerSec)}/s`;
}

/** Seconds → human-readable ETA. qBittorrent's sentinel for "unknown" is <0 or >=8640000. */
export function fmtEta(seconds: number): string {
	if (seconds < 0 || seconds >= 8640000) return '—';
	if (seconds < 60) return `${Math.round(seconds)}s`;
	if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
	const h = Math.floor(seconds / 3600);
	const m = Math.round((seconds % 3600) / 60);
	return `${h}h ${m}m`;
}

/** ISO timestamp → relative "just now" / "3h ago" / "2d ago" label. */
export function fmtRelativeTime(iso: string): string {
	const d = new Date(iso);
	const diffH = Math.round((Date.now() - d.getTime()) / 3600000);
	if (diffH < 1) return 'just now';
	if (diffH < 24) return `${diffH}h ago`;
	return `${Math.round(diffH / 24)}d ago`;
}

/** Rolls an episode's versions up to a single wanted-list status. */
export function episodeStatus(ep: UnifiedEpisode): 'missing' | 'upgradable' | 'queued' {
	if (ep.versions.some((v) => v.status === 'queued')) return 'queued';
	return ep.versions.some((v) => v.status === 'upgradable') ? 'upgradable' : 'missing';
}

/** Rolls an episode's versions up to the full status (used for row badges). */
export function fullEpisodeStatus(ep: UnifiedEpisode): EpisodeStatus {
	if (ep.versions.some((v) => v.status === 'importing')) return 'importing';
	if (ep.versions.some((v) => v.status === 'queued')) return 'queued';
	const imported = ep.versions.some((v) => v.status === 'imported');
	const upgradable = ep.versions.some((v) => v.status === 'upgradable');
	if (imported && upgradable) return 'upgradable';
	if (imported) return 'imported';
	return 'missing';
}

/** Short badge label for a version, e.g. "normal" -> "N", "extended" -> "EXT". */
export const versionBadgeLabel: Record<string, string> = {
	normal: 'N',
	extended: 'EXT'
};

/** Version labels (e.g. "normal", "extended") that currently have a file on disk for this episode. */
export function downloadedVersions(ep: UnifiedEpisode): string[] {
	return ep.versions.filter((v) => v.status === 'imported' || v.status === 'upgradable').map((v) => v.version);
}

/** Monitored episodes that are missing, upgradable, or queued — shared by the Wanted page and the sidebar badge count. */
export function wantedEpisodes(arcs: UnifiedArc[]) {
	return arcs.flatMap((arc) =>
		arc.episodes
			.filter(
				(ep) =>
					ep.monitored &&
					(!ep.downloaded || ep.versions.some((v) => v.status === 'upgradable' || v.status === 'queued'))
			)
			.map((ep) => ({ arc, ep, status: episodeStatus(ep) }))
	);
}

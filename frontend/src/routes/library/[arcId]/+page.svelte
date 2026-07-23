<!-- [arcId]/+page.svelte -->
<script lang="ts">
	import EpisodeDetailsModal from '$lib/components/EpisodeDetailsModal.svelte';
	import { Switch } from '$lib/components/ui/switch';

	import { page } from '$app/state';
	import { arcs } from '$lib/stores';
	import { api } from '$lib/api';
	import { episodeStatusMeta, fullEpisodeStatus, downloadedVersions, versionBadgeLabel } from '$lib/statusStyles';
	import { cn } from '$lib/utils';

	const arcId = $derived(page.params.arcId);

	let arcData = $state<UnifiedArc | null>(null);
	let selectedEpisode = $state<UnifiedEpisode | null>(null);
	let actionError = $state<string | null>(null);
	let actionMessage = $state<string | null>(null);

	let monitoringArc = $state(false);
	let downloadingMonitored = $state(false);
	let verifyingNFOs = $state(false);
	let downloadingCrc = $state<Set<string>>(new Set());

	$effect(() => {
		arcData = $arcs.find((a) => a.arc.toString() === arcId) ?? null;
	});

	async function refresh() {
		const list = await api.getAllEpisodes();
		arcs.set(list);
	}

	async function toggleMonitor(episode: UnifiedEpisode) {
		try {
			await api.toggleMonitor(episode.arc, episode.episode, !episode.monitored);
			await refresh();
		} catch (e) {
			actionError = e instanceof Error ? e.message : 'Failed to update monitor state';
		}
	}

	function actionFor(ep: UnifiedEpisode) {
		const status = fullEpisodeStatus(ep);
		if (status === 'imported') return null;
		if (status === 'importing')
			return { ...episodeStatusMeta.importing, label: 'Importing…', disabled: true };
		if (status === 'queued') return { ...episodeStatusMeta.queued, label: 'Queued', disabled: true };
		if (status === 'upgradable')
			return { ...episodeStatusMeta.upgradable, label: 'Upgrade', disabled: false };
		return { label: 'Download', disabled: false, color: '#4d9fff', bg: '#233042' };
	}

	async function downloadEpisode(ep: UnifiedEpisode) {
		const target = ep.versions.find((v) => v.status === 'missing' || v.status === 'upgradable');
		if (!target) return;
		downloadingCrc = new Set([...downloadingCrc, target.crc32]);
		actionError = null;
		try {
			await api.downloadEpisode(target.crc32);
			await refresh();
		} catch (e) {
			actionError = e instanceof Error ? e.message : 'Download failed';
		} finally {
			const next = new Set(downloadingCrc);
			next.delete(target.crc32);
			downloadingCrc = next;
		}
	}

	async function toggleMonitorArc() {
		if (!arcData) return;
		monitoringArc = true;
		actionError = null;
		actionMessage = null;
		try {
			await api.monitorArc(arcData.arc, !arcData.monitored);
			await refresh();
			actionMessage = arcData.monitored ? 'Arc unmonitored.' : 'All episodes monitored.';
		} catch (e) {
			actionError = e instanceof Error ? e.message : 'Failed to update arc monitor state';
		} finally {
			monitoringArc = false;
		}
	}

	async function downloadAllMonitored() {
		if (!arcData) return;
		downloadingMonitored = true;
		actionError = null;
		actionMessage = null;
		try {
			const result = await api.downloadMonitored(arcData.arc);
			actionMessage = `Queued ${result.queued} of ${result.total} monitored episode(s).`;
		} catch (e) {
			actionError = e instanceof Error ? e.message : 'Download failed';
		} finally {
			downloadingMonitored = false;
		}
	}

	async function verifyNFOs() {
		if (!arcData) return;
		verifyingNFOs = true;
		actionError = null;
		actionMessage = null;
		try {
			const result = await api.verifyNFOs(arcData.arc);
			actionMessage = `NFOs verified — ${result.updated} of ${result.total} updated.`;
		} catch (e) {
			actionError = e instanceof Error ? e.message : 'NFO verify failed';
		} finally {
			verifyingNFOs = false;
		}
	}

	const headerAction =
		'cursor-pointer rounded text-[11px] font-semibold px-2.5 py-1.5 transition-colors disabled:cursor-not-allowed disabled:opacity-50';
</script>

{#if arcData}
	<div>
		<!-- Arc header -->
		<div class="mb-4 rounded-md border border-border bg-card p-4">
			<div class="mb-3 flex flex-wrap items-start justify-between gap-2.5">
				<div>
					<div class="mb-1 flex items-center gap-2">
						<span class="text-[17px] font-bold text-card-foreground">{arcData.title}</span>
						{#if arcData.status}
							<span class="rounded-[2px] px-1.5 py-0.5 font-mono text-[10px]" style="color:#f5a623;background:#2a2213">
								{arcData.status}
							</span>
						{/if}
					</div>
					<div class="font-mono text-[11px] text-muted-foreground">
						{arcData.episodeCount} episodes
						{#if arcData.timeSavedMins}
							&middot; saves ~{arcData.timeSavedMins} min ({arcData.timeSavedPercent}%)
						{/if}
					</div>
				</div>
				<div class="flex flex-wrap gap-2">
					<button
						type="button"
						class={cn(headerAction, 'bg-secondary text-muted-foreground hover:text-card-foreground')}
						disabled={verifyingNFOs}
						onclick={verifyNFOs}
					>
						{verifyingNFOs ? 'Verifying…' : 'Verify NFOs'}
					</button>
					<button
						type="button"
						class={cn(headerAction, 'text-primary')}
						style="background:#233042"
						disabled={downloadingMonitored}
						onclick={downloadAllMonitored}
					>
						{downloadingMonitored ? 'Queueing…' : 'Download Monitored'}
					</button>
					<button
						type="button"
						class={cn(headerAction, 'font-bold')}
						style={arcData.monitored ? 'color:#8992a0;background:#20242b' : 'color:#4d9fff;background:#233042'}
						disabled={monitoringArc}
						onclick={toggleMonitorArc}
					>
						{monitoringArc ? 'Updating…' : arcData.monitored ? 'Unmonitor Arc' : 'Monitor Arc'}
					</button>
				</div>
			</div>

			<div class="grid grid-cols-2 gap-[11px] border-t border-border pt-3 text-[11.5px] sm:grid-cols-4">
				<div>
					<div class="mb-0.5 text-[10px] text-muted-foreground">MANGA CH.</div>
					<div class="font-mono text-card-foreground">
						{arcData.mangaChapters ?? '—'}{#if arcData.numberOfChapters} ({arcData.numberOfChapters}){/if}
					</div>
				</div>
				<div>
					<div class="mb-0.5 text-[10px] text-muted-foreground">ANIME EPS.</div>
					<div class="font-mono text-card-foreground">{arcData.animeEpisodes ?? '—'}</div>
				</div>
				<div>
					<div class="mb-0.5 text-[10px] text-muted-foreground">EPS. ADAPTED</div>
					<div class="font-mono text-card-foreground">{arcData.episodesAdapted ?? '—'}</div>
				</div>
				<div>
					<div class="mb-0.5 text-[10px] text-muted-foreground">FILLER EPS.</div>
					<div class="font-mono text-card-foreground">{arcData.fillerEpisodes || 'None'}</div>
				</div>
				<div>
					<div class="mb-0.5 text-[10px] text-muted-foreground">RESOLUTION</div>
					<div class="font-mono text-card-foreground">{arcData.resolution}</div>
				</div>
				<div>
					<div class="mb-0.5 text-[10px] text-muted-foreground">AUDIO</div>
					<div class="font-mono text-card-foreground">{arcData.audioLanguages}</div>
				</div>
				<div class="col-span-2">
					<div class="mb-0.5 text-[10px] text-muted-foreground">SUBTITLES</div>
					<div class="truncate font-mono text-card-foreground" title={arcData.subtitleLanguages}>
						{arcData.subtitleLanguages}
					</div>
				</div>
			</div>
		</div>

		{#if actionError}
			<p class="mb-3 text-xs text-destructive">{actionError}</p>
		{/if}
		{#if actionMessage}
			<p class="mb-3 text-xs" style="color:#3ecf8e">{actionMessage}</p>
		{/if}

		<!-- Episode table -->
		{#if arcData.episodes.length > 0}
			<div class="overflow-hidden rounded-md border border-border">
				<div
					class="grid gap-2 border-b border-border px-3.5 py-2 font-mono text-[10px] text-muted-foreground"
					style="grid-template-columns: 44px 1fr 100px 90px 130px 46px"
				>
					<div>EP</div>
					<div>TITLE</div>
					<div>RELEASED</div>
					<div>STATUS</div>
					<div>ACTION</div>
					<div></div>
				</div>
				{#each arcData.episodes as ep (ep.episode)}
					{@const status = fullEpisodeStatus(ep)}
					{@const meta = episodeStatusMeta[status]}
					{@const action = actionFor(ep)}
					{@const downloading = ep.versions.some((v) => downloadingCrc.has(v.crc32))}
						{@const versions = downloadedVersions(ep)}
					<div
						class="grid items-center gap-2 border-b border-[#20242c] px-3.5 py-2.5 text-[12.5px] last:border-b-0"
						style="grid-template-columns: 44px 1fr 100px 90px 130px 46px"
					>
						<div class="font-mono text-muted-foreground">{String(ep.episode).padStart(2, '0')}</div>
						<button
							type="button"
							class="cursor-pointer truncate text-left text-card-foreground"
							onclick={() => (selectedEpisode = ep)}
						>
							{ep.title}
						</button>
						<div class="font-mono text-[11px] text-muted-foreground">{ep.released}</div>
						<div class="flex items-center gap-1">
							<span class="rounded-[2px] px-1.5 py-0.5 font-mono text-[10px]" style="color:{meta.color};background:{meta.bg}">
								{meta.label}
							</span>
							{#each versions as v}
								<span
									class="rounded-[2px] px-1 py-0.5 font-mono text-[9px] text-muted-foreground"
									style="background:#20242b"
									title={v}
								>
									{versionBadgeLabel[v] ?? v}
								</span>
							{/each}
						</div>
						<div>
							{#if action}
								<button
									type="button"
									class="w-full cursor-pointer rounded py-1 text-center text-[11px] font-semibold disabled:cursor-not-allowed disabled:opacity-60"
									style="color:{action.color};background:{action.bg}"
									disabled={action.disabled || downloading}
									onclick={() => downloadEpisode(ep)}
								>
									{downloading ? 'Queuing…' : action.label}
								</button>
							{/if}
						</div>
						<div class="flex justify-end">
							<Switch
								size="sm"
								checked={ep.monitored}
								onCheckedChange={() => toggleMonitor(ep)}
							/>
						</div>
					</div>
				{/each}
			</div>
		{:else}
			<div class="py-12 text-center text-[12.5px] text-muted-foreground">
				No episodes released yet — this arc is being tracked and will populate automatically.
			</div>
		{/if}
	</div>
{/if}

<!-- MODAL -->
<EpisodeDetailsModal
	open={selectedEpisode !== null}
	onOpenChange={(open) => !open && (selectedEpisode = null)}
	episode={selectedEpisode}
/>

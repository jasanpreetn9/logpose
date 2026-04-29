<!-- [arcId]/+page.svelte -->
<script lang="ts">
	import EpisodeCard from '$lib/components/EpisodeCard.svelte';
	import EpisodeDetailsModal from '$lib/components/EpisodeDetailsModal.svelte';

	import { page } from '$app/state';
	import { arcs } from '$lib/stores';
	import { api } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import { CircleCheckBig, Download, FileCheck } from 'lucide-svelte';

	const arcId = $derived(page.params.arcId);

	let arcData = $state<UnifiedArc | null>(null);
	let selectedEpisode = $state<UnifiedEpisode | null>(null);
	let actionError = $state<string | null>(null);
	let actionMessage = $state<string | null>(null);

	let monitoringArc = $state(false);
	let downloadingMonitored = $state(false);
	let verifyingNFOs = $state(false);

	$effect(() => {
		arcData = $arcs.find((a) => a.arc.toString() === arcId) ?? null;
	});

	async function refresh() {
		const list = await api.getAllEpisodes();
		arcs.set(list);
	}

	const allMonitored = $derived(
		arcData != null && arcData.episodes.length > 0 && arcData.episodes.every((e) => e.monitored)
	);

	async function toggleMonitor(episode: UnifiedEpisode) {
		try {
			await api.toggleMonitor(episode.arc, episode.episode, !episode.monitored);
			await refresh();
		} catch (e) {
			actionError = e instanceof Error ? e.message : 'Failed to update monitor state';
		}
	}

	async function download(episode: UnifiedEpisode): Promise<void> {
		const target = episode.versions.find((v) => v.status === 'missing' || v.status === 'upgradable');
		if (!target) {
			throw new Error('No downloadable version available');
		}
		await api.downloadEpisode(target.crc32);
	}

	async function toggleMonitorArc() {
		if (!arcData) return;
		monitoringArc = true;
		actionError = null;
		actionMessage = null;
		try {
			await api.monitorArc(arcData.arc, !allMonitored);
			await refresh();
			actionMessage = allMonitored ? 'Arc unmonitored.' : 'All episodes monitored.';
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
</script>

{#if arcData}
	<div class="space-y-6 p-6">
		<!-- Arc header -->
		<div
			class="p-6 rounded-xl bg-card/40 backdrop-blur-sm shadow-sm hover:shadow-md transition border border-white/5"
		>
			<div class="flex justify-between gap-4 flex-wrap">
				<div class="space-y-3">
					<div class="flex items-center gap-3">
						<h2 class="text-2xl font-bold">Arc {arcData.arc}: {arcData.title}</h2>
						<span class="px-2 py-0.5 bg-primary/20 text-primary rounded text-sm">
							{arcData.status}
						</span>
					</div>

					<div class="text-sm text-muted-foreground flex gap-3 flex-wrap">
						<span>{arcData.episodeCount} Episodes</span>
						<span>•</span>
						<span>Audio: {arcData.audioLanguages}</span>
						<span>•</span>
						<span>Subtitles: {arcData.subtitleLanguages}</span>
						<span>•</span>
						<span>{arcData.resolution}</span>
					</div>
				</div>

				<div class="flex items-start gap-2 flex-wrap">
					<Button
						variant={allMonitored ? 'default' : 'outline'}
						size="sm"
						onclick={toggleMonitorArc}
						disabled={monitoringArc}
					>
						<CircleCheckBig class="mr-2 h-4 w-4" />
						{allMonitored ? 'Unmonitor Arc' : 'Monitor Arc'}
					</Button>

					<Button
						variant="outline"
						size="sm"
						onclick={downloadAllMonitored}
						disabled={downloadingMonitored}
					>
						<Download class="mr-2 h-4 w-4" />
						{downloadingMonitored ? 'Queueing…' : 'Download Monitored'}
					</Button>

					<Button
						variant="outline"
						size="sm"
						onclick={verifyNFOs}
						disabled={verifyingNFOs}
					>
						<FileCheck class="mr-2 h-4 w-4" />
						{verifyingNFOs ? 'Verifying…' : 'Verify NFOs'}
					</Button>
				</div>
			</div>
		</div>

		{#if actionError}
			<p class="text-sm text-red-500 px-1">{actionError}</p>
		{/if}
		{#if actionMessage}
			<p class="text-sm text-green-500 px-1">{actionMessage}</p>
		{/if}

		<!-- Episode list -->
		<div class="space-y-3">
			{#each arcData.episodes as episode}
				<EpisodeCard
					{episode}
					onSelect={() => (selectedEpisode = episode)}
					onToggleMonitor={() => toggleMonitor(episode)}
					onDownload={() => download(episode)}
				/>
			{/each}
		</div>
	</div>
{/if}

<!-- MODAL -->
<EpisodeDetailsModal
	open={selectedEpisode !== null}
	onOpenChange={(open) => !open && (selectedEpisode = null)}
	episode={selectedEpisode}
/>

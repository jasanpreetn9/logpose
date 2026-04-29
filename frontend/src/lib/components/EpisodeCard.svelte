<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import { Eye, Download, CircleCheckBig, CheckCheck } from 'lucide-svelte';

	interface Props {
		episode: UnifiedEpisode;
		onToggleMonitor?: () => void;
		onDownload?: () => Promise<void>;
		onSelect?: () => void;
	}

	let { episode, onToggleMonitor = () => {}, onDownload = async () => {}, onSelect = () => {} }: Props =
		$props();

	function getStatus(ep: UnifiedEpisode): 'imported' | 'missing' | 'upgradable' | 'none' {
		const imported = ep.versions.filter((v) => v.status === 'imported');
		const upgradable = ep.versions.filter((v) => v.status === 'upgradable');
		if (imported.length > 0 && upgradable.length > 0) return 'upgradable';
		if (imported.length > 0) return 'imported';
		if (ep.versions.length === 0) return 'none';
		return 'missing';
	}

	const downloadStatus = $derived(getStatus(episode));

	let downloading = $state(false);
	let queued = $state(false);
	let downloadError = $state<string | null>(null);
	let queuedTimer: ReturnType<typeof setTimeout> | null = null;

	async function handleDownload() {
		downloading = true;
		downloadError = null;
		queued = false;
		try {
			await onDownload();
			queued = true;
			if (queuedTimer) clearTimeout(queuedTimer);
			queuedTimer = setTimeout(() => (queued = false), 4000);
		} catch (e) {
			downloadError = e instanceof Error ? e.message : 'Download failed';
		} finally {
			downloading = false;
		}
	}
</script>

<div
	class="group border border-border bg-card rounded-lg p-5 hover:border-primary/50 hover:shadow-md transition-all"
>
	<div class="flex items-start justify-between gap-4">
		<div class="flex-1 space-y-2">
			<div class="flex items-center gap-3">
				<span class="text-sm font-medium text-muted-foreground">Episode {episode.episode}</span>
				<h3 class="text-base font-semibold">{episode.title}</h3>
			</div>

			<p class="text-sm text-muted-foreground">
				{episode.description || 'No description available.'}
			</p>

			{#each episode.versions as version}
				{#if version.status === 'imported'}
					<div class="flex items-center gap-4 text-xs text-muted-foreground">
						<span>Version: {version.version}</span>
						<span>CRC: {version.crc32}</span>
						<span>Status: {version.status}</span>
					</div>
					<div class="rounded bg-muted/50 px-3 py-2">
						<p class="font-mono text-xs text-muted-foreground">{version.file_path}</p>
					</div>
				{/if}
			{/each}

			{#if downloadError}
				<p class="text-xs text-red-500">{downloadError}</p>
			{/if}
		</div>

		<div class="flex flex-col items-end gap-2">
			<Button
				size="sm"
				variant={episode.monitored ? 'default' : 'outline'}
				onclick={onToggleMonitor}
			>
				<CircleCheckBig class="mr-2 h-4 w-4" />
				{episode.monitored ? 'Monitored' : 'Unmonitored'}
			</Button>

			{#if queued}
				<Button size="sm" variant="secondary" disabled>
					<CheckCheck class="mr-2 h-4 w-4 text-green-500" />
					Queued
				</Button>
			{:else if downloadStatus === 'missing'}
				<Button size="sm" variant="outline" onclick={handleDownload} disabled={downloading}>
					<Download class="mr-2 h-4 w-4" />
					{downloading ? 'Queuing…' : 'Download'}
				</Button>
			{:else if downloadStatus === 'upgradable'}
				<Button size="sm" variant="outline" onclick={handleDownload} disabled={downloading}>
					<Download class="mr-2 h-4 w-4" />
					{downloading ? 'Queuing…' : 'Upgrade'}
				</Button>
			{/if}

			<Button size="sm" variant="ghost" onclick={onSelect}>
				<Eye class="mr-2 h-4 w-4" />
				Details
			</Button>
		</div>
	</div>
</div>

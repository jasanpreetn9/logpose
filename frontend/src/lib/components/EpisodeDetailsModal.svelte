<script lang="ts">
	import { Dialog, DialogContent, DialogHeader, DialogTitle } from '$lib/components/ui/dialog';
	import { episodeStatusMeta } from '$lib/statusStyles';
	import { api } from '$lib/api';

	let {
		open,
		onOpenChange,
		episode,
		onDownloaded,
		onDeleted
	}: {
		open: boolean;
		onOpenChange: (open: boolean) => void;
		episode: UnifiedEpisode | null;
		onDownloaded?: () => void;
		onDeleted?: () => void;
	} = $props();

	let downloadingCrc = $state<Set<string>>(new Set());
	let deletingCrc = $state<Set<string>>(new Set());
	let armedCrc = $state<string | null>(null);
	let actionError = $state<string | null>(null);
	let armedTimer: ReturnType<typeof setTimeout> | null = null;

	async function downloadVersion(crc32: string) {
		downloadingCrc = new Set([...downloadingCrc, crc32]);
		actionError = null;
		try {
			await api.downloadEpisode(crc32);
			onDownloaded?.();
		} catch (e) {
			actionError = e instanceof Error ? e.message : 'Download failed';
		} finally {
			const next = new Set(downloadingCrc);
			next.delete(crc32);
			downloadingCrc = next;
		}
	}

	function handleDeleteClick(crc32: string) {
		if (armedCrc !== crc32) {
			armedCrc = crc32;
			if (armedTimer) clearTimeout(armedTimer);
			armedTimer = setTimeout(() => (armedCrc = null), 3000);
			return;
		}
		if (armedTimer) clearTimeout(armedTimer);
		armedCrc = null;
		deleteVersion(crc32);
	}

	async function deleteVersion(crc32: string) {
		deletingCrc = new Set([...deletingCrc, crc32]);
		actionError = null;
		try {
			await api.deleteEpisodeVersion(crc32);
			onDeleted?.();
		} catch (e) {
			actionError = e instanceof Error ? e.message : 'Delete failed';
		} finally {
			const next = new Set(deletingCrc);
			next.delete(crc32);
			deletingCrc = next;
		}
	}
</script>

<Dialog {open} {onOpenChange}>
	<DialogContent class="max-w-[520px] max-h-[78vh] overflow-auto bg-card">
		{#if episode}
			<DialogHeader>
				<DialogTitle class="text-[15px] text-card-foreground">
					{episode.title}
				</DialogTitle>
				<p class="font-mono text-[11px] text-muted-foreground">
					{episode.arc}.{String(episode.episode).padStart(2, '0')} &middot; Released {episode.released}
				</p>
			</DialogHeader>

			<p class="text-[12.5px] leading-relaxed text-[#a3a9b3]">
				{episode.description || 'No description available.'}
			</p>

			{#if actionError}
				<p class="text-[11.5px] text-destructive">{actionError}</p>
			{/if}

			<div>
				<div class="mb-1.5 font-mono text-[10px] text-muted-foreground">VERSIONS</div>
				<div class="flex flex-col gap-1.5">
					{#each episode.versions as version}
						{@const meta = episodeStatusMeta[version.status]}
						{@const canDownload = version.status === 'missing' || version.status === 'upgradable'}
						{@const downloading = downloadingCrc.has(version.crc32)}
						{@const deleting = deletingCrc.has(version.crc32)}
						{@const armed = armedCrc === version.crc32}
						<div class="rounded-[5px] border border-border bg-background p-2.5">
							<div class="mb-1 flex items-center justify-between gap-2">
								<div class="flex items-center gap-1.5">
									<span class="text-[12.5px] font-semibold capitalize text-card-foreground">{version.version}</span>
									<span class="rounded-[2px] px-1.5 py-0.5 font-mono text-[10px]" style="color:{meta.color};background:{meta.bg}">
										{meta.label}
									</span>
								</div>
								{#if canDownload}
									<button
										type="button"
										class="cursor-pointer rounded px-2 py-0.5 text-[10.5px] font-semibold text-primary disabled:cursor-not-allowed disabled:opacity-60"
										style="background:#233042"
										disabled={downloading}
										onclick={() => downloadVersion(version.crc32)}
									>
										{downloading ? 'Queuing…' : 'Download'}
									</button>
								{:else if version.file_path}
									<button
										type="button"
										class="cursor-pointer rounded px-2 py-0.5 text-[10.5px] font-semibold disabled:cursor-not-allowed disabled:opacity-60"
										style={armed
											? 'color:#e5484d;background:#2a1416'
											: 'color:#8992a0;background:#20242b'}
										disabled={deleting}
										onclick={() => handleDeleteClick(version.crc32)}
									>
										{deleting ? 'Deleting…' : armed ? 'Confirm?' : 'Delete'}
									</button>
								{/if}
							</div>
							<div class="font-mono text-[10.5px] text-muted-foreground">
								CRC32 {version.crc32} &middot; Released {version.released}
							</div>
							{#if version.file_path}
								<div class="mt-0.5 break-all font-mono text-[10px] text-muted-foreground">{version.file_path}</div>
							{/if}
						</div>
					{/each}
				</div>
			</div>
		{/if}
	</DialogContent>
</Dialog>

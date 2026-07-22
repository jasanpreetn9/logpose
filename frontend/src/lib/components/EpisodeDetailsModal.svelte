<script lang="ts">
	import { Dialog, DialogContent, DialogHeader, DialogTitle } from '$lib/components/ui/dialog';
	import { episodeStatusMeta } from '$lib/statusStyles';

	let {
		open,
		onOpenChange,
		episode
	}: {
		open: boolean;
		onOpenChange: (open: boolean) => void;
		episode: UnifiedEpisode | null;
	} = $props();
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

			<div>
				<div class="mb-1.5 font-mono text-[10px] text-muted-foreground">VERSIONS</div>
				<div class="flex flex-col gap-1.5">
					{#each episode.versions as version}
						{@const meta = episodeStatusMeta[version.status]}
						<div class="rounded-[5px] border border-border bg-background p-2.5">
							<div class="mb-1 flex items-center justify-between">
								<span class="text-[12.5px] font-semibold capitalize text-card-foreground">{version.version}</span>
								<span class="rounded-[2px] px-1.5 py-0.5 font-mono text-[10px]" style="color:{meta.color};background:{meta.bg}">
									{meta.label}
								</span>
							</div>
							<div class="font-mono text-[10.5px] text-muted-foreground">CRC32 {version.crc32}</div>
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

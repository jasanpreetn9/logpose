<script lang="ts">
	import { Dialog, DialogContent, DialogHeader, DialogTitle } from '$lib/components/ui/dialog';
	import { Badge } from '$lib/components/ui/badge';
	import { ScrollArea } from '$lib/components/ui/scroll-area';

	let {
		open,
		onOpenChange,
		episode
	}: {
		open: boolean;
		onOpenChange: (open: boolean) => void;
		episode: UnifiedEpisode | null;
	} = $props();

	const statusColors: Record<string, string> = {
		imported: 'bg-success text-success-foreground',
		missing: 'bg-destructive text-destructive-foreground',
		upgradable: 'bg-warning text-warning-foreground',
		downloading: 'bg-primary text-primary-foreground',
		queued: 'bg-primary/60 text-white',
		failed: 'bg-red-700 text-white',
		none: 'bg-muted text-muted-foreground'
	};

	function getStatus(ep: UnifiedEpisode): string {
		const imported = ep.versions.filter((v) => v.status === 'imported');
		const upgradable = ep.versions.filter((v) => v.status === 'upgradable');
		if (imported.length > 0 && upgradable.length > 0) return 'upgradable';
		if (imported.length > 0) return 'imported';
		if (ep.versions.length === 0) return 'none';
		return 'missing';
	}

	const downloadStatus = $derived(episode ? getStatus(episode) : 'none');
	const filePath = $derived(episode?.versions.find((v) => v.status === 'imported')?.file_path ?? null);
</script>

<Dialog {open} {onOpenChange}>
	<DialogContent class="max-w-3xl">
		{#if episode}
			<DialogHeader>
				<div class="flex items-center gap-3">
					<DialogTitle class="text-xl">
						Episode {episode.episode}: {episode.title}
					</DialogTitle>
					<Badge class={statusColors[downloadStatus]}>
						{downloadStatus}
					</Badge>
				</div>
			</DialogHeader>

			<ScrollArea class="max-h-[70vh] pr-4">
				<div class="space-y-6">
					<p class="text-muted-foreground">{episode.description}</p>

					<div class="grid grid-cols-3 gap-4 bg-muted/50 p-4 rounded-lg">
						<div>
							<p class="text-xs text-muted-foreground">Released</p>
							<p class="font-medium">{episode.released}</p>
						</div>

						{#if filePath}
							<div>
								<p class="text-xs text-muted-foreground">File</p>
								<p class="font-mono text-xs">{filePath}</p>
							</div>
						{/if}
					</div>
				</div>
			</ScrollArea>
		{/if}
	</DialogContent>
</Dialog>

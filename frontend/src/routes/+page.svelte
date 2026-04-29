<script lang="ts">
	import { arcs } from '$lib/stores';
	import { Badge } from '$lib/components/ui/badge';

	const totalEpisodes = $derived($arcs.reduce((sum, a) => sum + a.episodeCount, 0));
	const totalDownloaded = $derived($arcs.reduce((sum, a) => sum + a.episodesDownloaded, 0));
	const completeArcs = $derived($arcs.filter((a) => a.episodesDownloaded === a.episodeCount && a.episodeCount > 0).length);
</script>

<div class="p-6 space-y-6">
	<div>
		<h1 class="text-2xl font-bold">Dashboard</h1>
		<p class="text-muted-foreground text-sm mt-1">One Pace library overview</p>
	</div>

	<div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
		<div class="rounded-xl border bg-card p-4 space-y-1">
			<p class="text-xs text-muted-foreground uppercase tracking-wider">Arcs</p>
			<p class="text-2xl font-bold">{$arcs.length}</p>
		</div>
		<div class="rounded-xl border bg-card p-4 space-y-1">
			<p class="text-xs text-muted-foreground uppercase tracking-wider">Episodes</p>
			<p class="text-2xl font-bold">{totalEpisodes}</p>
		</div>
		<div class="rounded-xl border bg-card p-4 space-y-1">
			<p class="text-xs text-muted-foreground uppercase tracking-wider">Downloaded</p>
			<p class="text-2xl font-bold">{totalDownloaded}</p>
		</div>
		<div class="rounded-xl border bg-card p-4 space-y-1">
			<p class="text-xs text-muted-foreground uppercase tracking-wider">Complete Arcs</p>
			<p class="text-2xl font-bold">{completeArcs}</p>
		</div>
	</div>

	<div>
		<h2 class="text-lg font-semibold mb-3">Arcs</h2>
		<div class="space-y-2">
			{#each $arcs as arc}
				<a
					href={`/library/${arc.arc}`}
					class="flex items-center justify-between rounded-lg border bg-card px-4 py-3 hover:border-primary/50 transition"
				>
					<div class="flex items-center gap-3">
						<span class="text-sm text-muted-foreground w-6">#{arc.arc}</span>
						<span class="text-sm font-medium">{arc.title}</span>
					</div>
					<div class="flex items-center gap-3">
						<div class="h-1.5 w-24 rounded-full bg-muted overflow-hidden">
							<div
								class="h-full rounded-full bg-primary transition-all"
								style="width: {arc.episodeCount > 0 ? (arc.episodesDownloaded / arc.episodeCount) * 100 : 0}%"
							></div>
						</div>
						<Badge
							variant="secondary"
							class={arc.episodesDownloaded === arc.episodeCount && arc.episodeCount > 0
								? 'bg-green-600 text-white'
								: ''}
						>
							{arc.episodesDownloaded}/{arc.episodeCount}
						</Badge>
					</div>
				</a>
			{/each}
		</div>
	</div>
</div>

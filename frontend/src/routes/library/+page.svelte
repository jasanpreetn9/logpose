<script lang="ts">
	import { arcs } from '$lib/stores';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';

	const list = $derived($arcs);

	const totalEpisodes = $derived(list.reduce((sum, a) => sum + a.episodeCount, 0));
	const totalDownloaded = $derived(list.reduce((sum, a) => sum + a.episodesDownloaded, 0));
	const completeArcs = $derived(list.filter((a) => a.episodesDownloaded === a.episodeCount && a.episodeCount > 0).length);
	const monitored = $derived(list.reduce((sum, a) => sum + a.episodes.filter((e) => e.monitored).length, 0));
</script>

<div class="p-6 space-y-6">
	<!-- Stats -->
	<div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
		<div class="rounded-xl border bg-card p-4 space-y-1">
			<p class="text-xs text-muted-foreground uppercase tracking-wider">Arcs</p>
			<p class="text-2xl font-bold">{list.length}</p>
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
			<p class="text-xs text-muted-foreground uppercase tracking-wider">Monitored</p>
			<p class="text-2xl font-bold">{monitored}</p>
		</div>
	</div>

	<!-- Arc grid -->
	<div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 2xl:grid-cols-5">
		{#each list as arc}
			<a
				href={`/library/${arc.arc}`}
				class="group block rounded-xl border bg-card hover:shadow-lg hover:border-primary transition p-5"
			>
				<!-- Header -->
				<div class="flex justify-between items-start mb-2">
					<div>
						<h2 class="text-lg font-semibold group-hover:text-primary transition">
							Arc {arc.arc}: {arc.title}
						</h2>
						<p class="text-xs text-muted-foreground">{arc.resolution}</p>
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

				<!-- Progress bar -->
				<div class="h-1.5 w-full rounded-full bg-muted overflow-hidden mb-3">
					<div
						class="h-full rounded-full bg-primary transition-all"
						style="width: {arc.episodeCount > 0 ? (arc.episodesDownloaded / arc.episodeCount) * 100 : 0}%"
					></div>
				</div>

				<!-- Metadata -->
				<div class="text-sm text-muted-foreground space-y-1">
					<p><span class="font-medium text-foreground">Audio:</span> {arc.audioLanguages}</p>
					<p><span class="font-medium text-foreground">Subtitles:</span> {arc.subtitleLanguages}</p>

					{#if arc.mangaChapters !== null}
						<p><span class="font-medium text-foreground">Manga Chapters:</span> {arc.mangaChapters}</p>
					{/if}
					{#if arc.timeSavedMins !== null}
						<p><span class="font-medium text-foreground">Time Saved:</span> {arc.timeSavedMins} min ({arc.timeSavedPercent}%)</p>
					{/if}
				</div>

				<!-- Footer -->
				<div class="mt-4 flex justify-between items-center">
					<Badge
						variant="outline"
						class={arc.episodesDownloaded === arc.episodeCount
							? 'border-green-600 text-green-600'
							: arc.episodesDownloaded > 0
								? 'border-yellow-500 text-yellow-600'
								: 'border-red-500 text-red-600'}
					>
						{arc.episodesDownloaded === arc.episodeCount
							? 'Complete'
							: arc.episodesDownloaded > 0
								? 'In Progress'
								: 'Not Started'}
					</Badge>

					<Button size="sm" variant="ghost" class="opacity-0 group-hover:opacity-100 transition">
						Open →
					</Button>
				</div>
			</a>
		{/each}
	</div>
</div>

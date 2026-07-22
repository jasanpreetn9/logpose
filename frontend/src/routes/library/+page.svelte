<script lang="ts">
	import { arcs } from '$lib/stores';

	const list = $derived($arcs);

	function thumbGradient(arc: UnifiedArc, pct: number) {
		if (arc.status) return 'linear-gradient(135deg,#3a2f1c,#1c2028)';
		if (pct === 0) return 'linear-gradient(135deg,#2a2028,#1c2028)';
		return 'linear-gradient(135deg,#233042,#1c2028)';
	}

	function numColor(arc: UnifiedArc, pct: number) {
		if (arc.status) return '#f5a623';
		if (pct === 0) return '#8992a0';
		return '#4d9fff';
	}

	function countColor(pct: number) {
		if (pct >= 100) return '#3ecf8e';
		if (pct > 0) return '#f5a623';
		return '#6b7280';
	}

	function subsShort(subtitleLanguages: string) {
		const langs = subtitleLanguages
			.split(',')
			.map((s) => s.trim())
			.filter(Boolean);
		return langs.length > 3 ? `${langs.slice(0, 3).join(', ')} +${langs.length - 3}` : subtitleLanguages;
	}
</script>

<div class="grid gap-3" style="grid-template-columns: repeat(auto-fill, minmax(250px, 1fr))">
	{#each list as arc (arc.arc)}
		{@const pct = arc.episodeCount > 0 ? (arc.episodesDownloaded / arc.episodeCount) * 100 : 0}
		<a
			href={`/library/${arc.arc}`}
			class="cursor-pointer rounded-[5px] border border-border bg-card p-[13px] transition-colors hover:border-[#3a4150]"
		>
			<div
				class="mb-2.5 flex h-14 items-center justify-center rounded-[3px] font-mono text-[12px] font-semibold"
				style="background:{thumbGradient(arc, pct)};color:{numColor(arc, pct)}"
			>
				{String(arc.arc).padStart(2, '0')}
			</div>

			<div class="mb-1.5 flex items-center gap-1.5">
				<div class="flex-1 text-[13.5px] font-semibold text-card-foreground">{arc.title}</div>
				{#if arc.status}
					<span class="rounded-[2px] px-1.5 py-0.5 font-mono text-[9.5px]" style="color:#f5a623;background:#2a2213">
						{arc.status}
					</span>
				{/if}
			</div>

			<div class="mb-2 flex flex-wrap gap-1.5 font-mono text-[10px] text-muted-foreground">
				<span class="rounded-[2px] bg-secondary px-1.5 py-0.5">{arc.resolution}</span>
				<span class="rounded-[2px] bg-secondary px-1.5 py-0.5">{arc.audioLanguages}</span>
				<span class="rounded-[2px] bg-secondary px-1.5 py-0.5" title={arc.subtitleLanguages}>
					{subsShort(arc.subtitleLanguages)}
				</span>
			</div>

			<div class="mb-1 flex items-center justify-between">
				<span class="font-mono text-[10.5px]" style="color:{countColor(pct)}">
					{arc.episodesDownloaded}/{arc.episodeCount} eps
				</span>
				<span class="font-mono text-[10.5px] text-[#8992a0]">{Math.round(pct)}%</span>
			</div>
			<div class="h-[3px] overflow-hidden rounded-full bg-[#2a2e36]">
				<div class="h-full" style="width:{pct}%;background:{countColor(pct)}"></div>
			</div>
		</a>
	{/each}
</div>

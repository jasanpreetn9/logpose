<script lang="ts">
	import { arcs } from '$lib/stores';
	import { api } from '$lib/api';
	import { wantedEpisodes, episodeStatusMeta } from '$lib/statusStyles';

	const wanted = $derived(wantedEpisodes($arcs));

	let downloading = $state<Set<string>>(new Set());
	let errors = $state<Map<string, string>>(new Map());

	function actionStyle(status: 'missing' | 'upgradable' | 'queued') {
		if (status === 'upgradable') return episodeStatusMeta.upgradable;
		if (status === 'queued') return episodeStatusMeta.queued;
		return { color: '#4d9fff', bg: '#233042' };
	}

	async function download(crc32: string) {
		downloading = new Set([...downloading, crc32]);
		errors = new Map([...errors].filter(([k]) => k !== crc32));
		try {
			await api.downloadEpisode(crc32);
			arcs.set(await api.getAllEpisodes());
		} catch (e) {
			errors = new Map([...errors, [crc32, e instanceof Error ? e.message : 'Download failed']]);
		} finally {
			const next = new Set(downloading);
			next.delete(crc32);
			downloading = next;
		}
	}
</script>

{#if wanted.length === 0}
	<div class="py-14 text-center text-[13px] text-muted-foreground">
		Nothing wanted — all monitored episodes are up to date.
	</div>
{:else}
	<div class="overflow-hidden rounded-md border border-border">
		<div
			class="grid gap-2 border-b border-border px-3.5 py-2 font-mono text-[10px] text-muted-foreground"
			style="grid-template-columns: 1fr 90px 130px"
		>
			<div>EPISODE</div>
			<div>STATUS</div>
			<div>ACTION</div>
		</div>
		{#each wanted as { arc, ep, status } (`${arc.arc}-${ep.episode}`)}
			{@const meta = episodeStatusMeta[status]}
			{@const action = actionStyle(status)}
			{@const target = ep.versions.find((v) => v.status === status)}
			<div
				class="grid items-center gap-2 border-b border-[#20242c] px-3.5 py-2.5 text-[12.5px] last:border-b-0"
				style="grid-template-columns: 1fr 90px 130px"
			>
				<div class="truncate">
					<span class="text-card-foreground">{ep.title}</span>
					<span class="ml-2 font-mono text-[11px] text-muted-foreground">
						{arc.title} &middot; {String(ep.episode).padStart(2, '0')}
					</span>
				</div>
				<div>
					<span class="rounded-[2px] px-1.5 py-0.5 font-mono text-[10px]" style="color:{meta.color};background:{meta.bg}">
						{meta.label}
					</span>
				</div>
				<div>
					{#if status === 'queued'}
						<button
							type="button"
							class="w-full cursor-not-allowed rounded py-1.5 text-center text-[11px] font-semibold opacity-60"
							style="color:{action.color};background:{action.bg}"
							disabled
						>
							Queued
						</button>
					{:else if target}
						<button
							type="button"
							class="w-full cursor-pointer rounded py-1.5 text-center text-[11px] font-semibold disabled:cursor-not-allowed disabled:opacity-60"
							style="color:{action.color};background:{action.bg}"
							disabled={downloading.has(target.crc32)}
							onclick={() => download(target.crc32)}
						>
							{downloading.has(target.crc32) ? 'Queuing…' : status === 'upgradable' ? 'Upgrade' : 'Download'}
						</button>
						{#if errors.has(target.crc32)}
							<p class="mt-1 text-[10.5px] text-destructive">{errors.get(target.crc32)}</p>
						{/if}
					{/if}
				</div>
			</div>
		{/each}
	</div>
{/if}

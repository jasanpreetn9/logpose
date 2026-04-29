<script lang="ts">
	import { arcs } from '$lib/stores';
	import { api } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import { Download, ArrowUpFromLine } from 'lucide-svelte';

	function epStatus(ep: UnifiedEpisode): 'missing' | 'upgradable' {
		return ep.versions.some((v) => v.status === 'upgradable') ? 'upgradable' : 'missing';
	}

	const wanted = $derived(
		$arcs.flatMap((arc) =>
			arc.episodes
				.filter((ep) => ep.monitored && (!ep.downloaded || ep.versions.some((v) => v.status === 'upgradable')))
				.map((ep) => ({ arc, ep, status: epStatus(ep) }))
		)
	);

	let downloading = $state<Set<string>>(new Set());
	let errors = $state<Map<string, string>>(new Map());

	async function download(crc32: string) {
		downloading = new Set([...downloading, crc32]);
		errors = new Map([...errors].filter(([k]) => k !== crc32));
		try {
			await api.downloadEpisode(crc32);
		} catch (e) {
			errors = new Map([...errors, [crc32, e instanceof Error ? e.message : 'Download failed']]);
		} finally {
			const next = new Set(downloading);
			next.delete(crc32);
			downloading = next;
		}
	}
</script>

<div class="p-6 space-y-4">
	<div>
		<h1 class="text-2xl font-bold">Wanted</h1>
		<p class="text-muted-foreground text-sm mt-1">Monitored episodes missing or with available upgrades</p>
	</div>

	{#if wanted.length === 0}
		<p class="text-muted-foreground">No missing or upgradable monitored episodes.</p>
	{:else}
		<div class="space-y-2">
			{#each wanted as { arc, ep, status }}
				{@const target = ep.versions.find((v) => v.status === status)}
				<div class="flex items-center justify-between rounded-lg border bg-card px-4 py-3">
					<div class="space-y-0.5">
						<div class="flex items-center gap-2">
							<Badge variant="secondary" class="text-xs">Arc {arc.arc}</Badge>
							<span class="text-sm font-medium">{ep.title}</span>
							{#if status === 'upgradable'}
								<Badge variant="outline" class="text-xs border-yellow-500 text-yellow-600">Upgrade Available</Badge>
							{/if}
						</div>
						<p class="text-xs text-muted-foreground">Episode {ep.episode} · {arc.title}</p>
					</div>
					<div class="flex flex-col items-end gap-1">
						{#if target}
							<Button
								size="sm"
								variant={status === 'upgradable' ? 'default' : 'outline'}
								disabled={downloading.has(target.crc32)}
								onclick={() => download(target.crc32)}
							>
								{#if status === 'upgradable'}
									<ArrowUpFromLine class="mr-2 h-4 w-4" />
									{downloading.has(target.crc32) ? 'Queuing…' : 'Upgrade'}
								{:else}
									<Download class="mr-2 h-4 w-4" />
									{downloading.has(target.crc32) ? 'Queuing…' : 'Download'}
								{/if}
							</Button>
						{/if}
						{#if target && errors.has(target.crc32)}
							<p class="text-xs text-red-500">{errors.get(target.crc32)}</p>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

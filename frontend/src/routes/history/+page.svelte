<script lang="ts">
	import { api } from '$lib/api';
	import { historyEvents } from '$lib/stores';
	import { Button } from '$lib/components/ui/button';
	import { RefreshCw, Film } from 'lucide-svelte';

	let refreshing = $state(false);

	async function refresh() {
		refreshing = true;
		try {
			historyEvents.set(await api.getHistory());
		} finally {
			refreshing = false;
		}
	}

	function formatTime(ts: string): string {
		return new Date(ts).toLocaleString();
	}
</script>

<div class="p-6 space-y-4">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold">History</h1>
			<p class="text-muted-foreground text-sm mt-1">All imported episodes</p>
		</div>
		<Button variant="outline" size="sm" onclick={refresh} disabled={refreshing}>
			<RefreshCw class="mr-2 h-4 w-4 {refreshing ? 'animate-spin' : ''}" />
			Refresh
		</Button>
	</div>

	{#if $historyEvents.length === 0}
		<p class="text-muted-foreground text-sm">
			No imports yet. Use "Scan Downloads" to import episodes from your downloads folder.
		</p>
	{:else}
		<div class="space-y-2">
			{#each $historyEvents as ev}
				<div class="flex items-start gap-3 rounded-lg border bg-card px-4 py-3">
					<Film class="mt-0.5 h-4 w-4 shrink-0 text-primary" />
					<div class="flex-1 min-w-0 space-y-0.5">
						<p class="text-sm font-medium">{ev.message.replace(/^Imported:\s*/, '')}</p>
						{#if ev.details}
							<p class="text-xs text-muted-foreground font-mono truncate" title={ev.details}>
								{ev.details}
							</p>
						{/if}
					</div>
					<span class="text-xs text-muted-foreground shrink-0">{formatTime(ev.timestamp)}</span>
				</div>
			{/each}
		</div>
	{/if}
</div>

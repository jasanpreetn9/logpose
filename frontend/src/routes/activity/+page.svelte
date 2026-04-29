<script lang="ts">
	import { api } from '$lib/api';
	import { activity } from '$lib/stores';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { RefreshCw, CheckCircle, XCircle } from 'lucide-svelte';

	let refreshing = $state(false);

	async function refresh() {
		refreshing = true;
		try {
			activity.set(await api.getActivity());
		} finally {
			refreshing = false;
		}
	}

	function eventLabel(type: ActivityEvent['type']): string {
		const map: Record<ActivityEvent['type'], string> = {
			download_queued: 'Download',
			download_failed: 'Download',
			library_scan: 'Library Scan',
			downloads_scan: 'Downloads Scan',
			import: 'Import'
		};
		return map[type] ?? type;
	}

	function eventVariant(ev: ActivityEvent): 'default' | 'secondary' | 'destructive' | 'outline' {
		if (!ev.success) return 'destructive';
		if (ev.type === 'import') return 'default';
		return 'secondary';
	}

	function formatTime(ts: string): string {
		return new Date(ts).toLocaleString();
	}
</script>

<div class="p-6 space-y-4">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold">Activity</h1>
			<p class="text-muted-foreground text-sm mt-1">Recent download and scan events</p>
		</div>
		<Button variant="outline" size="sm" onclick={refresh} disabled={refreshing}>
			<RefreshCw class="mr-2 h-4 w-4 {refreshing ? 'animate-spin' : ''}" />
			Refresh
		</Button>
	</div>

	{#if $activity.length === 0}
		<p class="text-muted-foreground text-sm">No activity yet. Try scanning your library or downloads.</p>
	{:else}
		<div class="space-y-2">
			{#each $activity as ev}
				<div class="flex items-start gap-3 rounded-lg border bg-card px-4 py-3">
					{#if ev.success}
						<CheckCircle class="mt-0.5 h-4 w-4 shrink-0 text-green-500" />
					{:else}
						<XCircle class="mt-0.5 h-4 w-4 shrink-0 text-red-500" />
					{/if}
					<div class="flex-1 min-w-0 space-y-0.5">
						<div class="flex items-center gap-2 flex-wrap">
							<Badge variant={eventVariant(ev)} class="text-xs">{eventLabel(ev.type)}</Badge>
							<span class="text-sm font-medium">{ev.message}</span>
						</div>
						{#if ev.details}
							<p class="text-xs text-muted-foreground font-mono truncate">{ev.details}</p>
						{/if}
					</div>
					<span class="text-xs text-muted-foreground shrink-0">{formatTime(ev.timestamp)}</span>
				</div>
			{/each}
		</div>
	{/if}
</div>

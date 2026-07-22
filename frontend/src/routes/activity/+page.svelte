<script lang="ts">
	import { api } from '$lib/api';
	import { activity } from '$lib/stores';
	import { activityTypeStyle, fmtRelativeTime } from '$lib/statusStyles';

	let refreshing = $state(false);

	async function refresh() {
		refreshing = true;
		try {
			activity.set(await api.getActivity());
		} finally {
			refreshing = false;
		}
	}
</script>

<div class="mb-3 flex items-center justify-end">
	<button
		type="button"
		class="cursor-pointer rounded border border-border bg-card px-2.5 py-1.5 text-[11px] font-semibold text-muted-foreground transition-colors hover:text-card-foreground disabled:opacity-50"
		disabled={refreshing}
		onclick={refresh}
	>
		{refreshing ? 'Refreshing…' : 'Refresh'}
	</button>
</div>

{#if $activity.length === 0}
	<div class="py-12 text-center text-[12.5px] text-muted-foreground">
		No activity yet. Try scanning your library or downloads.
	</div>
{:else}
	<div class="flex flex-col gap-px overflow-hidden rounded-md border border-border bg-border">
		{#each $activity as ev (ev.id)}
			{@const meta = activityTypeStyle(ev.type)}
			<div class="flex items-start gap-3 bg-card px-4 py-2.5">
				<span class="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full" style="background:{ev.success ? meta.color : '#e5484d'}"></span>
				<span class="w-[110px] shrink-0 pt-px font-mono text-[9.5px]" style="color:{ev.success ? meta.color : '#e5484d'}">
					{meta.label}
				</span>
				<div class="min-w-0 flex-1">
					<div class="text-[12.5px] text-card-foreground">{ev.message}</div>
					{#if ev.details}
						<div class="mt-0.5 truncate font-mono text-[10.5px] text-muted-foreground" title={ev.details}>
							{ev.details}
						</div>
					{/if}
				</div>
				<span class="shrink-0 font-mono text-[10px] text-muted-foreground">{fmtRelativeTime(ev.timestamp)}</span>
			</div>
		{/each}
	</div>
{/if}

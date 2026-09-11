<script lang="ts">
	import type { ComponentType, SvelteComponent } from 'svelte';

	let {
		icon: Icon,
		title,
		description,
		actionLabel,
		actionHref
	}: {
		icon: ComponentType<SvelteComponent>;
		title: string;
		description?: string;
		actionLabel?: string;
		actionHref?: string;
	} = $props();
</script>

<div class="flex flex-col items-center gap-3 py-16 text-center">
	<div
		class="flex h-11 w-11 items-center justify-center rounded-full bg-secondary text-muted-foreground"
	>
		<Icon class="h-5 w-5" />
	</div>
	<div class="text-[13px] font-semibold text-card-foreground">{title}</div>
	{#if description}
		<div class="max-w-[320px] text-[12px] text-muted-foreground">{description}</div>
	{/if}
	{#if actionLabel && actionHref}
		<a
			href={actionHref}
			class="mt-1 cursor-pointer rounded px-3 py-1.5 text-[11.5px] font-semibold text-primary transition-colors hover:brightness-110"
			style="background:#233042"
		>
			{actionLabel}
		</a>
	{/if}
</div>

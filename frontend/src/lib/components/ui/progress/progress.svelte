<script lang="ts">
	import { Progress as ProgressPrimitive } from "bits-ui";
	import { cn } from "$lib/utils.js";

	let {
		ref = $bindable(null),
		class: className,
		indicatorClass,
		indicatorColor,
		value = 0,
		max = 100,
		...restProps
	}: ProgressPrimitive.RootProps & {
		indicatorClass?: string;
		indicatorColor?: string;
	} = $props();

	const percent = $derived(Math.min(100, Math.max(0, ((value ?? 0) / (max || 100)) * 100)));
</script>

<ProgressPrimitive.Root
	bind:ref
	{value}
	{max}
	data-slot="progress"
	class={cn("relative h-[3px] w-full overflow-hidden rounded-full bg-[#2a2e36]", className)}
	{...restProps}
>
	<div
		data-slot="progress-indicator"
		class={cn("h-full rounded-full transition-[width]", indicatorClass)}
		style="width: {percent}%; background: {indicatorColor ?? 'var(--primary)'}"
	></div>
</ProgressPrimitive.Root>

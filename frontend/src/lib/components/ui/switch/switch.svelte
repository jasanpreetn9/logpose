<script lang="ts">
	import { Switch as SwitchPrimitive } from "bits-ui";
	import { cn, type WithoutChildrenOrChild } from "$lib/utils.js";

	let {
		ref = $bindable(null),
		checked = $bindable(false),
		class: className,
		size = "default",
		...restProps
	}: WithoutChildrenOrChild<SwitchPrimitive.RootProps> & { size?: "default" | "sm" } = $props();

	const trackClass = size === "sm" ? "h-[13px] w-6" : "h-4 w-[30px]";
	const thumbClass =
		size === "sm"
			? "size-[9px] translate-x-0.5 data-[state=checked]:translate-x-[13px]"
			: "size-3 translate-x-0.5 data-[state=checked]:translate-x-[18px]";
</script>

<SwitchPrimitive.Root
	bind:ref
	bind:checked
	data-slot="switch"
	class={cn(
		"peer inline-flex shrink-0 cursor-pointer items-center rounded-full border-none transition-colors outline-none disabled:cursor-not-allowed disabled:opacity-50",
		"data-[state=checked]:bg-primary data-[state=unchecked]:bg-[#2a2e36]",
		trackClass,
		className
	)}
	{...restProps}
>
	<SwitchPrimitive.Thumb
		data-slot="switch-thumb"
		class={cn("pointer-events-none block rounded-full bg-sidebar transition-transform", thumbClass)}
	/>
</SwitchPrimitive.Root>

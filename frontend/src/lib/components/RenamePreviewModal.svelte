<script lang="ts">
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogTitle,
		DialogDescription
	} from '$lib/components/ui/dialog';
	import { ScrollArea } from '$lib/components/ui/scroll-area';
	import { MoveRight } from 'lucide-svelte';

	let {
		open,
		onOpenChange,
		renames,
		total,
		renaming,
		onConfirm
	}: {
		open: boolean;
		onOpenChange: (open: boolean) => void;
		renames: RenamePreviewItem[];
		total: number;
		renaming: boolean;
		onConfirm: () => void;
	} = $props();

	const folders = $derived(
		[...new Set(renames.map((r) => r.folder))].map((folder) => ({
			folder,
			items: renames.filter((r) => r.folder === folder)
		}))
	);
</script>

<Dialog {open} {onOpenChange}>
	<DialogContent class="max-w-[640px] max-h-[74vh] overflow-auto bg-card">
		<DialogHeader>
			<DialogTitle class="text-[14px] text-card-foreground">Rename Files</DialogTitle>
			<DialogDescription class="text-[11.5px] text-muted-foreground">
				{#if renames.length === 0}
					All {total} recognized files already have the correct name.
				{:else}
					{renames.length} of {total} recognized files will be renamed. Sidecar .nfo and thumbnail
					files are moved along with each video.
				{/if}
			</DialogDescription>
		</DialogHeader>

		{#if renames.length > 0}
			<ScrollArea class="max-h-[50vh] pr-4">
				<div class="flex flex-col gap-3">
					{#each folders as group}
						<div>
							<p class="mb-1 font-mono text-[10px] text-muted-foreground">{group.folder}</p>
							<div class="flex flex-col gap-1.5">
								{#each group.items as item}
									<div class="rounded-[5px] border border-border bg-background px-2.5 py-2 font-mono text-[10.5px]">
										<p class="mb-0.5 break-all text-destructive line-through">{item.from}</p>
										<p class="flex items-center gap-1.5 break-all" style="color:#3ecf8e">
											<MoveRight class="h-3 w-3 shrink-0" />
											{item.to}
										</p>
									</div>
								{/each}
							</div>
						</div>
					{/each}
				</div>
			</ScrollArea>
		{/if}

		{#if renames.length > 0}
			<button
				type="button"
				class="w-full cursor-pointer rounded-[5px] py-2 text-center text-[12.5px] font-bold text-primary transition-colors hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-60"
				style="background:#233042"
				disabled={renaming}
				onclick={onConfirm}
			>
				{renaming ? 'Renaming…' : 'Confirm Rename'}
			</button>
		{/if}
	</DialogContent>
</Dialog>

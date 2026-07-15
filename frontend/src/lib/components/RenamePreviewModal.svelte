<script lang="ts">
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogTitle,
		DialogDescription,
		DialogFooter
	} from '$lib/components/ui/dialog';
	import { Button } from '$lib/components/ui/button';
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
	<DialogContent class="max-w-4xl">
		<DialogHeader>
			<DialogTitle>Rename Files</DialogTitle>
			<DialogDescription>
				{#if renames.length === 0}
					All {total} recognized files already have the correct name.
				{:else}
					{renames.length} of {total} recognized files will be renamed. Sidecar .nfo and thumbnail
					files are moved along with each video.
				{/if}
			</DialogDescription>
		</DialogHeader>

		{#if renames.length > 0}
			<ScrollArea class="max-h-[60vh] pr-4">
				<div class="space-y-4">
					{#each folders as group}
						<div>
							<p class="mb-1 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
								{group.folder}
							</p>
							<div class="space-y-1">
								{#each group.items as item}
									<div class="rounded-md border bg-muted/30 px-3 py-2">
										<p class="font-mono text-xs text-muted-foreground line-through">{item.from}</p>
										<p class="flex items-center gap-1.5 font-mono text-xs text-foreground">
											<MoveRight class="h-3 w-3 shrink-0 text-green-500" />
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

		<DialogFooter>
			<Button variant="outline" onclick={() => onOpenChange(false)} disabled={renaming}>
				Cancel
			</Button>
			{#if renames.length > 0}
				<Button onclick={onConfirm} disabled={renaming}>
					{renaming ? 'Renaming…' : `Rename ${renames.length} File${renames.length === 1 ? '' : 's'}`}
				</Button>
			{/if}
		</DialogFooter>
	</DialogContent>
</Dialog>

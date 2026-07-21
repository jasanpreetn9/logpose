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
	import { MoveRight } from 'lucide-svelte';

	let {
		open,
		onOpenChange,
		preview,
		loading,
		error,
		importing,
		onConfirm
	}: {
		open: boolean;
		onOpenChange: (open: boolean) => void;
		preview: ManualImportPreview | null;
		loading: boolean;
		error: string | null;
		importing: boolean;
		onConfirm: () => void;
	} = $props();
</script>

<Dialog {open} {onOpenChange}>
	<DialogContent class="max-w-2xl">
		<DialogHeader>
			<DialogTitle>Import File</DialogTitle>
			<DialogDescription>
				{#if loading}
					Resolving metadata…
				{:else if preview}
					This is what will be applied — confirm the episode is correct before importing.
				{/if}
			</DialogDescription>
		</DialogHeader>

		{#if error}
			<p class="text-sm text-red-500">{error}</p>
		{:else if preview}
			<div class="space-y-3">
				<div class="rounded-md border bg-muted/30 px-3 py-2 text-sm">
					<span class="text-muted-foreground">Arc {preview.arc}</span>
					· Episode {preview.episode}
					<span class="text-muted-foreground">({preview.version})</span>
					<p class="mt-1 font-medium">{preview.title}</p>
				</div>

				<div class="rounded-md border bg-muted/30 px-3 py-2">
					<p class="text-xs text-muted-foreground mb-1">{preview.destFolder}</p>
					<p class="flex items-center gap-1.5 font-mono text-xs text-foreground">
						<MoveRight class="h-3 w-3 shrink-0 text-green-500" />
						{preview.destFilename}
					</p>
				</div>
			</div>
		{/if}

		<DialogFooter>
			<Button variant="outline" onclick={() => onOpenChange(false)} disabled={importing}>
				Cancel
			</Button>
			{#if preview}
				<Button onclick={onConfirm} disabled={importing || loading}>
					{importing ? 'Importing…' : 'Import File'}
				</Button>
			{/if}
		</DialogFooter>
	</DialogContent>
</Dialog>

<script lang="ts">
	import { Dialog, DialogContent, DialogHeader, DialogTitle } from '$lib/components/ui/dialog';

	let {
		open,
		onOpenChange,
		file,
		arcs,
		preview,
		loading,
		error,
		importing,
		onPreview,
		onConfirm
	}: {
		open: boolean;
		onOpenChange: (open: boolean) => void;
		file: UnmatchedFile | null;
		arcs: UnifiedArc[];
		preview: ManualImportPreview | null;
		loading: boolean;
		error: string | null;
		importing: boolean;
		onPreview: (params: { arc: number; episode: number; version: string }) => void;
		onConfirm: (params: { arc: number; episode: number; version: string }) => void;
	} = $props();

	let selectedArc = $state<number | null>(null);
	let selectedEpisode = $state<number | null>(null);
	let selectedVersion = $state<'normal' | 'extended'>('normal');

	// Reset the picker to the first arc/episode whenever a new file is opened for matching.
	$effect(() => {
		if (open && file) {
			selectedArc = arcs[0]?.arc ?? null;
			selectedEpisode = arcs[0]?.episodes[0]?.episode ?? null;
			selectedVersion = 'normal';
		}
	});

	const episodeOptions = $derived(arcs.find((a) => a.arc === selectedArc)?.episodes ?? []);

	function handleArcChange(e: Event) {
		selectedArc = Number((e.target as HTMLSelectElement).value);
		selectedEpisode = episodeOptions[0]?.episode ?? null;
	}

	function submitPreview() {
		if (selectedArc == null || selectedEpisode == null) return;
		onPreview({ arc: selectedArc, episode: selectedEpisode, version: selectedVersion });
	}

	function submitConfirm() {
		if (selectedArc == null || selectedEpisode == null) return;
		onConfirm({ arc: selectedArc, episode: selectedEpisode, version: selectedVersion });
	}

	const selectClass =
		'block w-full box-border rounded border border-[#2a2f3a] bg-background px-2.5 py-2 text-[12px] text-card-foreground';
</script>

<Dialog {open} {onOpenChange}>
	<DialogContent class="max-w-[480px] bg-card">
		<DialogHeader>
			<DialogTitle class="text-card-foreground">Match Download</DialogTitle>
		</DialogHeader>

		{#if file}
			<div class="mb-3.5 truncate font-mono text-[10.5px] text-muted-foreground">{file.name}</div>
		{/if}

		{#if error}
			<p class="text-sm text-destructive">{error}</p>
		{/if}

		{#if preview}
			<div class="mb-3.5 rounded border border-border bg-background p-3 text-[12px]">
				<div class="mb-1.5 font-semibold text-card-foreground">{preview.title}</div>
				<div class="mb-0.5 truncate font-mono text-[10.5px] text-destructive line-through">{file?.name}</div>
				<div class="truncate font-mono text-[10.5px]" style="color:#3ecf8e">
					{preview.destFolder}/{preview.destFilename}
				</div>
			</div>
			<button
				type="button"
				class="w-full cursor-pointer rounded py-2 text-center text-[12.5px] font-bold text-primary transition-colors hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-60"
				style="background:#233042"
				disabled={importing}
				onclick={submitConfirm}
			>
				{importing ? 'Importing…' : 'Confirm Import'}
			</button>
		{:else}
			<div class="flex flex-col gap-2.5">
				<label class="text-[11px] text-muted-foreground">
					Arc
					<select class={selectClass + ' mt-1.5'} value={selectedArc} onchange={handleArcChange}>
						{#each arcs as arc}
							<option value={arc.arc}>{arc.title}</option>
						{/each}
					</select>
				</label>
				<label class="text-[11px] text-muted-foreground">
					Episode
					<select class={selectClass + ' mt-1.5'} bind:value={selectedEpisode}>
						{#each episodeOptions as ep}
							<option value={ep.episode}>{ep.title}</option>
						{/each}
					</select>
				</label>
				<label class="text-[11px] text-muted-foreground">
					Version
					<select class={selectClass + ' mt-1.5'} bind:value={selectedVersion}>
						<option value="normal">normal</option>
						<option value="extended">extended</option>
					</select>
				</label>
				<button
					type="button"
					class="mt-1 w-full cursor-pointer rounded py-2 text-center text-[12.5px] font-bold text-primary transition-colors hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-60"
					style="background:#233042"
					disabled={loading || selectedArc == null || selectedEpisode == null}
					onclick={submitPreview}
				>
					{loading ? 'Resolving…' : 'Preview'}
				</button>
			</div>
		{/if}
	</DialogContent>
</Dialog>

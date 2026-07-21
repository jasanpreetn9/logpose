<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { api } from '$lib/api';
	import { arcs } from '$lib/stores';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import ManualImportModal from '$lib/components/ManualImportModal.svelte';
	import { X, FileQuestion } from 'lucide-svelte';

	let queue = $state<QueueItem[]>([]);
	let loaded = $state(false);
	let error = $state<string | null>(null);
	let removing = $state<Set<string>>(new Set());

	let unmatched = $state<UnmatchedFile[]>([]);
	let pickerOpenFor = $state<string | null>(null);
	let pickerArc = $state<number | null>(null);
	let pickerEpisode = $state<number | null>(null);
	let pickerVersion = $state<'normal' | 'extended'>('normal');

	let modalOpen = $state(false);
	let modalPreview = $state<ManualImportPreview | null>(null);
	let modalLoading = $state(false);
	let modalError = $state<string | null>(null);
	let modalImporting = $state(false);
	let modalTarget = $state<UnmatchedFile | null>(null);

	let timer: ReturnType<typeof setInterval> | null = null;

	async function load() {
		try {
			queue = await api.getQueue();
			error = null;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load queue';
		} finally {
			loaded = true;
		}
	}

	async function loadUnmatched() {
		try {
			unmatched = await api.getUnmatchedFiles();
		} catch {
			// Non-critical — leave the section empty rather than surfacing an error.
		}
	}

	onMount(() => {
		load();
		loadUnmatched();
		timer = setInterval(load, 5000);
	});

	onDestroy(() => {
		if (timer) clearInterval(timer);
	});

	function openPicker(file: UnmatchedFile) {
		pickerOpenFor = file.path;
		pickerArc = $arcs[0]?.arc ?? null;
		pickerEpisode = $arcs[0]?.episodes[0]?.episode ?? null;
		pickerVersion = 'normal';
	}

	const pickerEpisodes = $derived($arcs.find((a) => a.arc === pickerArc)?.episodes ?? []);

	async function openPreview(file: UnmatchedFile) {
		if (pickerArc == null || pickerEpisode == null) return;
		modalTarget = file;
		modalPreview = null;
		modalError = null;
		modalLoading = true;
		modalOpen = true;
		pickerOpenFor = null;
		try {
			modalPreview = await api.previewManualImport({
				path: file.path,
				arc: pickerArc,
				episode: pickerEpisode,
				version: pickerVersion
			});
		} catch (e) {
			modalError = e instanceof Error ? e.message : 'Failed to resolve metadata';
		} finally {
			modalLoading = false;
		}
	}

	async function confirmImport() {
		if (!modalTarget || pickerArc == null || pickerEpisode == null) return;
		modalImporting = true;
		try {
			await api.confirmManualImport({
				path: modalTarget.path,
				arc: pickerArc,
				episode: pickerEpisode,
				version: pickerVersion
			});
			unmatched = unmatched.filter((f) => f.path !== modalTarget?.path);
			arcs.set(await api.getAllEpisodes());
			modalOpen = false;
		} catch (e) {
			modalError = e instanceof Error ? e.message : 'Import failed';
		} finally {
			modalImporting = false;
		}
	}

	async function remove(hash: string) {
		removing = new Set([...removing, hash]);
		try {
			await api.removeFromQueue(hash);
			queue = queue.filter((q) => q.hash !== hash);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Remove failed';
		} finally {
			const next = new Set(removing);
			next.delete(hash);
			removing = next;
		}
	}

	function formatSize(bytes: number): string {
		if (!bytes) return '—';
		const units = ['B', 'KiB', 'MiB', 'GiB'];
		let i = 0;
		let v = bytes;
		while (v >= 1024 && i < units.length - 1) {
			v /= 1024;
			i++;
		}
		return `${v.toFixed(v >= 100 ? 0 : 1)} ${units[i]}`;
	}

	function formatSpeed(bps: number): string {
		return bps > 0 ? `${formatSize(bps)}/s` : '—';
	}

	function formatETA(seconds: number): string {
		if (!seconds || seconds < 0 || seconds >= 8640000) return '—';
		const h = Math.floor(seconds / 3600);
		const m = Math.floor((seconds % 3600) / 60);
		const s = seconds % 60;
		if (h > 0) return `${h}h ${m}m`;
		if (m > 0) return `${m}m ${s}s`;
		return `${s}s`;
	}

	const stateColors: Record<string, string> = {
		downloading: 'bg-primary text-primary-foreground',
		stalledDL: 'bg-warning text-warning-foreground',
		uploading: 'bg-green-600 text-white',
		stalledUP: 'bg-green-600/60 text-white',
		pausedDL: 'bg-muted text-muted-foreground',
		error: 'bg-destructive text-destructive-foreground'
	};
</script>

<div class="p-6 space-y-4">
	<div>
		<h1 class="text-2xl font-bold">Queue</h1>
		<p class="text-muted-foreground text-sm mt-1">Downloads currently in qBittorrent</p>
	</div>

	{#if error}
		<p class="text-sm text-red-500">{error}</p>
	{/if}

	{#if loaded && queue.length === 0}
		<p class="text-muted-foreground">Queue is empty.</p>
	{:else if queue.length > 0}
		<div class="space-y-2">
			{#each queue as item (item.hash)}
				<div class="rounded-lg border bg-card px-4 py-3">
					<div class="flex items-center justify-between gap-4">
						<div class="min-w-0 flex-1 space-y-1.5">
							<div class="flex items-center gap-2">
								{#if item.title}
									<Badge variant="secondary" class="text-xs shrink-0">
										Arc {item.arc} · Ep {item.episode}
									</Badge>
									<span class="truncate text-sm font-medium">{item.title}</span>
								{:else}
									<span class="truncate text-sm font-medium">{item.name}</span>
								{/if}
								<Badge class={`text-xs shrink-0 ${stateColors[item.state] ?? 'bg-muted text-muted-foreground'}`}>
									{item.state}
								</Badge>
							</div>

							<div class="h-2 w-full overflow-hidden rounded-full bg-muted">
								<div
									class="h-full rounded-full bg-primary transition-all"
									style={`width: ${Math.round(item.progress * 100)}%`}
								></div>
							</div>

							<div class="flex items-center gap-4 text-xs text-muted-foreground">
								<span>{Math.round(item.progress * 100)}%</span>
								<span>{formatSize(item.size)}</span>
								<span>{formatSpeed(item.dlspeed)}</span>
								<span>ETA {formatETA(item.eta)}</span>
							</div>
						</div>

						<Button
							size="sm"
							variant="ghost"
							disabled={removing.has(item.hash)}
							onclick={() => remove(item.hash)}
							title="Remove from qBittorrent (keeps files)"
						>
							<X class="h-4 w-4" />
						</Button>
					</div>
				</div>
			{/each}
		</div>
	{/if}

	{#if unmatched.length > 0}
		<div class="pt-4">
			<div class="flex items-center gap-2 mb-2">
				<FileQuestion class="h-4 w-4 text-muted-foreground" />
				<h2 class="text-lg font-semibold">Unmatched Downloads</h2>
			</div>
			<p class="text-muted-foreground text-sm mb-3">
				Files Logpose couldn't automatically match to an episode. Pick the correct episode to
				import them manually.
			</p>
			<div class="space-y-2">
				{#each unmatched as file (file.path)}
					<div class="rounded-lg border bg-card px-4 py-3 space-y-2">
						<div class="flex items-center justify-between gap-4">
							<div class="min-w-0 flex-1">
								<p class="truncate text-sm font-medium">{file.name}</p>
								<p class="text-xs text-muted-foreground">
									{file.reason === 'unparseable'
										? "Filename doesn't match a One Pace release"
										: `Unknown CRC ${file.crc32}`}
								</p>
							</div>
							<Button
								size="sm"
								variant="outline"
								onclick={() => (pickerOpenFor === file.path ? (pickerOpenFor = null) : openPicker(file))}
							>
								Match
							</Button>
						</div>

						{#if pickerOpenFor === file.path}
							<div class="flex flex-wrap items-center gap-2 border-t pt-2">
								<select
									class="rounded-md border bg-background px-2 py-1 text-sm"
									bind:value={pickerArc}
									onchange={() => (pickerEpisode = pickerEpisodes[0]?.episode ?? null)}
								>
									{#each $arcs as arc}
										<option value={arc.arc}>Arc {arc.arc}: {arc.title}</option>
									{/each}
								</select>
								<select class="rounded-md border bg-background px-2 py-1 text-sm" bind:value={pickerEpisode}>
									{#each pickerEpisodes as ep}
										<option value={ep.episode}>Ep {ep.episode}: {ep.title}</option>
									{/each}
								</select>
								<select class="rounded-md border bg-background px-2 py-1 text-sm" bind:value={pickerVersion}>
									<option value="normal">Normal</option>
									<option value="extended">Extended</option>
								</select>
								<Button size="sm" onclick={() => openPreview(file)}>Preview</Button>
							</div>
						{/if}
					</div>
				{/each}
			</div>
		</div>
	{/if}
</div>

<ManualImportModal
	open={modalOpen}
	onOpenChange={(o) => (modalOpen = o)}
	preview={modalPreview}
	loading={modalLoading}
	error={modalError}
	importing={modalImporting}
	onConfirm={confirmImport}
/>

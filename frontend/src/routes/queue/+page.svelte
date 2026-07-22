<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { api } from '$lib/api';
	import { arcs } from '$lib/stores';
	import ManualImportModal from '$lib/components/ManualImportModal.svelte';
	import { queueStateStyle, fmtBytes, fmtSpeed, fmtEta } from '$lib/statusStyles';

	let queue = $state<QueueItem[]>([]);
	let loaded = $state(false);
	let error = $state<string | null>(null);
	let removing = $state<Set<string>>(new Set());

	let unmatched = $state<UnmatchedFile[]>([]);

	let modalOpen = $state(false);
	let modalTarget = $state<UnmatchedFile | null>(null);
	let modalPreview = $state<ManualImportPreview | null>(null);
	let modalLoading = $state(false);
	let modalError = $state<string | null>(null);
	let modalImporting = $state(false);

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

	function openMatch(file: UnmatchedFile) {
		modalTarget = file;
		modalPreview = null;
		modalError = null;
		modalOpen = true;
	}

	async function previewMatch(params: { arc: number; episode: number; version: string }) {
		if (!modalTarget) return;
		modalPreview = null;
		modalError = null;
		modalLoading = true;
		try {
			modalPreview = await api.previewManualImport({ path: modalTarget.path, ...params });
		} catch (e) {
			modalError = e instanceof Error ? e.message : 'Failed to resolve metadata';
		} finally {
			modalLoading = false;
		}
	}

	async function confirmMatch(params: { arc: number; episode: number; version: string }) {
		if (!modalTarget) return;
		modalImporting = true;
		try {
			await api.confirmManualImport({ path: modalTarget.path, ...params });
			const path = modalTarget.path;
			unmatched = unmatched.filter((f) => f.path !== path);
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

	function queueLabel(item: QueueItem): string {
		if (!item.title) return item.name;
		const arc = $arcs.find((a) => a.arc === item.arc);
		return arc ? `${arc.title} — ${item.title}` : item.title;
	}
</script>

{#if error}
	<p class="mb-3 text-xs text-destructive">{error}</p>
{/if}

{#if loaded && queue.length === 0}
	<div class="py-10 text-center text-[13px] text-muted-foreground">Queue is empty</div>
{:else if queue.length > 0}
	<div class="mb-6 flex flex-col gap-2.5">
		{#each queue as item (item.hash)}
			{@const state = queueStateStyle(item.state)}
			{@const pct = Math.round(item.progress * 100)}
			<div class="rounded-md border border-border bg-card px-4 py-3.5">
				<div class="mb-2 flex items-center justify-between gap-2">
					<div class="min-w-0">
						<div class="truncate text-[13px] font-semibold text-card-foreground">{queueLabel(item)}</div>
						<div class="mt-0.5 truncate font-mono text-[10px] text-muted-foreground">{item.name}</div>
					</div>
					<div class="flex shrink-0 items-center gap-3.5">
						<span class="rounded px-1.5 py-0.5 font-mono text-[10.5px]" style="color:{state.color};background:{state.bg}">
							{state.label}
						</span>
						<button
							type="button"
							class="cursor-pointer text-[14px] text-muted-foreground hover:text-card-foreground disabled:opacity-50"
							title="Remove torrent (keeps files)"
							disabled={removing.has(item.hash)}
							onclick={() => remove(item.hash)}
						>
							&times;
						</button>
					</div>
				</div>
				<div class="mb-1.5 h-[5px] overflow-hidden rounded-full bg-[#262b34]">
					<div class="h-full transition-[width] duration-300" style="width:{pct}%;background:{state.color}"></div>
				</div>
				<div class="flex justify-between font-mono text-[10.5px] text-muted-foreground">
					<span>{pct}%</span><span>{fmtSpeed(item.dlspeed)}</span><span>ETA {fmtEta(item.eta)}</span><span>{fmtBytes(item.size)}</span>
				</div>
			</div>
		{/each}
	</div>
{/if}

<div class="mb-2.5 text-[12.5px] font-bold text-card-foreground">Unmatched Downloads</div>
{#if unmatched.length === 0}
	<div class="text-[12px] text-muted-foreground">Nothing unmatched.</div>
{:else}
	<div class="flex flex-col gap-2">
		{#each unmatched as file (file.path)}
			<div class="flex items-center justify-between gap-2.5 rounded-md border border-border bg-card px-4 py-2.5">
				<div class="min-w-0">
					<div class="truncate font-mono text-[12.5px] text-card-foreground">{file.name}</div>
					<div class="mt-0.5 text-[10.5px] text-muted-foreground">
						{file.reason === 'unparseable' ? "Filename doesn't match a One Pace release" : 'Unknown CRC'}
						{#if file.crc32}&middot; CRC {file.crc32}{/if}
					</div>
				</div>
				<button
					type="button"
					class="shrink-0 cursor-pointer rounded px-3 py-1.5 text-[11px] font-semibold text-primary transition-colors hover:brightness-110"
					style="background:#233042"
					onclick={() => openMatch(file)}
				>
					Match…
				</button>
			</div>
		{/each}
	</div>
{/if}

<ManualImportModal
	open={modalOpen}
	onOpenChange={(o) => (modalOpen = o)}
	file={modalTarget}
	arcs={$arcs}
	preview={modalPreview}
	loading={modalLoading}
	error={modalError}
	importing={modalImporting}
	onPreview={previewMatch}
	onConfirm={confirmMatch}
/>

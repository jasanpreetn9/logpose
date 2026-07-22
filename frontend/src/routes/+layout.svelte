<!-- +layout -->
<script lang="ts">
	let { children } = $props();
	import { page } from '$app/state';
	import { onMount, onDestroy } from 'svelte';
	import '$lib/app.css';
	import { Menu, X } from 'lucide-svelte';
	import RenamePreviewModal from '$lib/components/RenamePreviewModal.svelte';

	import { arcs, sidebarOpen, activity, historyEvents } from '$lib/stores';
	import { navigationItems } from '$lib';
	import { cn } from '$lib/utils';
	import { api } from '$lib/api';
	import { startSSE } from '$lib/sse';
	import { wantedEpisodes, qbitColors, type QbitStatus } from '$lib/statusStyles';

	const pathname = $derived(page.url.pathname);

	let scanningLibrary = $state(false);
	let scanningDownloads = $state(false);
	let refreshingMetadata = $state(false);
	let renamingFiles = $state(false);
	let scanError = $state<string | null>(null);
	let toast = $state<string | null>(null);
	let toastTimer: ReturnType<typeof setTimeout> | null = null;
	let appVersion = $state<string | null>(null);

	let healthChecks = $state<HealthCheck[]>([]);
	let dismissedHealthIds = $state<Set<string>>(new Set());
	const visibleHealthChecks = $derived(healthChecks.filter((c) => !dismissedHealthIds.has(c.id)));

	let qbEnabled = $state(true);
	const qbitStatus = $derived<QbitStatus>(
		!qbEnabled ? 'disabled' : healthChecks.some((c) => c.id === 'qbittorrent') ? 'error' : 'connected'
	);
	const qbitStatusLabel = $derived(
		{ disabled: 'QBIT DISABLED', error: 'QBIT ERROR', connected: 'QBIT CONNECTED', testing: 'QBIT TESTING' }[
			qbitStatus
		]
	);

	let queueCount = $state(0);
	const wantedCount = $derived(wantedEpisodes($arcs).length);

	const activeArcId = $derived(pathname.match(/^\/library\/(\d+)/)?.[1] ?? null);
	const activeArcTitle = $derived($arcs.find((a) => String(a.arc) === activeArcId)?.title ?? null);
	const breadcrumb = $derived(
		activeArcId
			? (activeArcTitle ?? `Arc ${activeArcId}`)
			: (navigationItems.find((item) => pathname.startsWith(item.href))?.label ?? 'Library')
	);
	const showBack = $derived(activeArcId !== null);

	let stopSSE: (() => void) | null = null;
	let healthTimer: ReturnType<typeof setInterval> | null = null;
	let queueTimer: ReturnType<typeof setInterval> | null = null;

	async function loadHealth() {
		try {
			const res = await api.getHealth();
			healthChecks = res.checks;
			dismissedHealthIds = new Set(
				[...dismissedHealthIds].filter((id) => res.checks.some((c) => c.id === id))
			);
		} catch {
			// Health checks are best-effort — don't surface fetch failures as a check.
		}
	}

	async function loadQueueCount() {
		try {
			queueCount = (await api.getQueue()).length;
		} catch {
			// Best-effort — the Queue page itself surfaces fetch failures.
		}
	}

	async function loadQbEnabled() {
		try {
			qbEnabled = (await api.getConfig()).qbEnabled;
		} catch {
			// Best-effort — leave the last-known state on failure.
		}
	}

	function showToast(message: string) {
		toast = message;
		if (toastTimer) clearTimeout(toastTimer);
		toastTimer = setTimeout(() => (toast = null), 4000);
	}

	onMount(async () => {
		const [list, acts, hist] = await Promise.all([
			api.getAllEpisodes(),
			api.getActivity(),
			api.getHistory()
		]);
		arcs.set(list);
		activity.set(acts);
		historyEvents.set(hist);

		api.getVersion()
			.then((v) => (appVersion = v))
			.catch(() => (appVersion = null));

		loadHealth();
		loadQbEnabled();
		healthTimer = setInterval(() => {
			loadHealth();
			loadQbEnabled();
		}, 60000);

		loadQueueCount();
		queueTimer = setInterval(loadQueueCount, 5000);

		stopSSE = startSSE((ev) => {
			activity.update((l) => [ev, ...l]);
			if (ev.type === 'import') {
				historyEvents.update((l) => [ev, ...l]);
			}
		});
	});

	onDestroy(() => {
		stopSSE?.();
		if (healthTimer) clearInterval(healthTimer);
		if (queueTimer) clearInterval(queueTimer);
		if (toastTimer) clearTimeout(toastTimer);
	});

	function dismissHealthCheck(id: string) {
		dismissedHealthIds = new Set([...dismissedHealthIds, id]);
	}

	async function handleScanLibrary() {
		scanningLibrary = true;
		scanError = null;
		try {
			await api.scanLibrary();
			arcs.set(await api.getAllEpisodes());
		} catch (e) {
			scanError = e instanceof Error ? e.message : 'Scan failed';
		} finally {
			scanningLibrary = false;
		}
	}

	let renameModalOpen = $state(false);
	let renamePreview = $state<RenamePreviewItem[]>([]);
	let renamePreviewTotal = $state(0);
	let loadingRenamePreview = $state(false);

	async function handleRenameFiles() {
		loadingRenamePreview = true;
		scanError = null;
		try {
			const preview = await api.previewRename();
			renamePreview = preview.renames;
			renamePreviewTotal = preview.total;
			renameModalOpen = true;
		} catch (e) {
			scanError = e instanceof Error ? e.message : 'Rename preview failed';
		} finally {
			loadingRenamePreview = false;
		}
	}

	async function confirmRename() {
		renamingFiles = true;
		scanError = null;
		try {
			const result = await api.renameFiles();
			arcs.set(await api.getAllEpisodes());
			showToast(`Renamed ${result.renamed} of ${result.total} files.`);
			renameModalOpen = false;
		} catch (e) {
			scanError = e instanceof Error ? e.message : 'Rename failed';
			renameModalOpen = false;
		} finally {
			renamingFiles = false;
		}
	}

	async function handleRefreshMetadata() {
		refreshingMetadata = true;
		scanError = null;
		try {
			const result = await api.refreshMetadata();
			arcs.set(await api.getAllEpisodes());
			showToast(
				result.grabbed > 0
					? `Metadata refreshed — ${result.grabbed} episode(s) auto-grabbed.`
					: 'Metadata refreshed.'
			);
		} catch (e) {
			scanError = e instanceof Error ? e.message : 'Refresh failed';
		} finally {
			refreshingMetadata = false;
		}
	}

	async function handleScanDownloads() {
		scanningDownloads = true;
		scanError = null;
		try {
			await api.scanDownloads();
			arcs.set(await api.getAllEpisodes());
		} catch (e) {
			scanError = e instanceof Error ? e.message : 'Scan failed';
		} finally {
			scanningDownloads = false;
		}
	}

	const toolbarChip =
		'cursor-pointer rounded text-[11px] font-semibold text-[#8992a0] bg-card border border-border px-2.5 py-1.5 transition-colors hover:text-foreground hover:border-[#3a4150] disabled:cursor-not-allowed disabled:opacity-50';
</script>

<!-- Mobile Sidebar Overlay -->
{#if $sidebarOpen}
	<button
		type="button"
		aria-label="Close sidebar"
		class="fixed inset-0 z-40 bg-black/50 lg:hidden"
		onclick={() => sidebarOpen.set(false)}
	></button>
{/if}

<div class="flex h-screen overflow-hidden text-[13px]">
	<!-- Sidebar -->
	<aside
		class={cn(
			'fixed inset-y-0 left-0 z-50 flex w-[184px] flex-none transform flex-col gap-0.5 bg-sidebar border-r border-sidebar-border py-[18px] transition-transform duration-200 lg:relative lg:translate-x-0',
			$sidebarOpen ? 'translate-x-0' : '-translate-x-full'
		)}
	>
		<div class="flex items-center justify-between px-[18px] pb-[18px]">
			<span class="font-mono text-[15px] font-bold tracking-wide text-card-foreground">
				LOG<span class="text-primary">POSE</span>
			</span>
			<button
				type="button"
				aria-label="Close sidebar"
				class="text-muted-foreground lg:hidden"
				onclick={() => sidebarOpen.set(false)}
			>
				<X class="h-4 w-4" />
			</button>
		</div>

		<nav class="flex flex-1 flex-col gap-0.5">
			{#each navigationItems as item}
				{@const active = pathname.startsWith(item.href)}
				<a
					href={item.href}
					onclick={() => sidebarOpen.set(false)}
					class={cn(
						'flex cursor-pointer items-center justify-between gap-2 border-l-2 px-[18px] py-[9px] text-[12.5px] font-semibold transition-colors',
						active ? 'border-primary text-card-foreground bg-[#161a20]' : 'border-transparent text-[#8992a0] hover:text-card-foreground'
					)}
				>
					<span>{item.label}</span>
					{#if item.href === '/wanted' && wantedCount > 0}
						<span
							class="font-mono text-[10px] rounded-full px-1.5 py-px"
							style="background:#3a2a13;color:#f5a623"
						>
							{wantedCount}
						</span>
					{:else if item.href === '/queue' && queueCount > 0}
						<span
							class="font-mono text-[10px] rounded-full px-1.5 py-px"
							style="background:#233042;color:#4d9fff"
						>
							{queueCount}
						</span>
					{/if}
				</a>
			{/each}
		</nav>

		<div class="mt-auto px-[18px] pt-4">
			<span class="font-mono text-[10px] text-[#4b5057]">LOGPOSE {appVersion ?? '…'}</span>
		</div>
	</aside>

	<!-- Main Content -->
	<div class="flex flex-1 flex-col overflow-hidden">
		<header
			class="flex min-h-[52px] flex-none flex-wrap items-center justify-between gap-2 border-b border-border px-[22px]"
		>
			<div class="flex items-center gap-2 py-2.5 font-mono text-[12px] font-semibold text-card-foreground">
				<button type="button" aria-label="Open sidebar" class="text-muted-foreground lg:hidden" onclick={() => sidebarOpen.set(true)}>
					<Menu class="h-4 w-4" />
				</button>
				{#if showBack}
					<a href="/library" class="text-primary hover:text-[#7ab6ff]">&larr;</a>
				{/if}
				{breadcrumb}
			</div>

			<div class="flex flex-wrap items-center gap-2 py-2">
				<button type="button" class={toolbarChip} disabled={scanningLibrary} onclick={handleScanLibrary}>
					{scanningLibrary ? 'Scanning…' : 'Scan Library'}
				</button>
				<button type="button" class={toolbarChip} disabled={scanningDownloads} onclick={handleScanDownloads}>
					{scanningDownloads ? 'Scanning…' : 'Scan Downloads'}
				</button>
				<button type="button" class={toolbarChip} disabled={refreshingMetadata} onclick={handleRefreshMetadata}>
					{refreshingMetadata ? 'Refreshing…' : 'Refresh Metadata'}
				</button>
				<button type="button" class={toolbarChip} disabled={loadingRenamePreview || renamingFiles} onclick={handleRenameFiles}>
					{loadingRenamePreview ? 'Checking…' : 'Rename Files'}
				</button>
				<span class="ml-1.5 flex items-center gap-1.5 font-mono text-[10.5px]" style="color:{qbitColors[qbitStatus]}">
					<span class="inline-block h-1.5 w-1.5 rounded-full" style="background:{qbitColors[qbitStatus]}"></span>
					{qbitStatusLabel}
				</span>
			</div>
		</header>

		{#if scanError}
			<div class="flex-none border-b border-border px-[22px] py-1.5 text-xs text-destructive">
				{scanError}
			</div>
		{/if}

		{#if visibleHealthChecks.length > 0}
			<div class="flex-none">
				{#each visibleHealthChecks as check (check.id)}
					<div
						class="flex items-center justify-between gap-4 border-b px-[22px] py-2 text-[12px]"
						style={check.level === 'error'
							? 'background:#2a1416;color:#e5484d;border-color:#3a1c1e'
							: 'background:#2a2213;color:#f5a623;border-color:#3a2f18'}
					>
						<span>
							<b class="mr-2 font-mono text-[9.5px] tracking-wide">{check.level.toUpperCase()}</b>
							{check.message}
						</span>
						<button
							type="button"
							class="cursor-pointer opacity-70 hover:opacity-100"
							onclick={() => dismissHealthCheck(check.id)}
							aria-label="Dismiss"
						>
							&times;
						</button>
					</div>
				{/each}
			</div>
		{/if}

		{#if toast}
			<div class="flex-none border-b px-[22px] py-2.5 text-[12px]" style="background:#122019;color:#3ecf8e;border-color:#1c3327">
				{toast}
			</div>
		{/if}

		<main class="flex-1 overflow-auto p-[22px]">
			{@render children()}
		</main>

		<RenamePreviewModal
			open={renameModalOpen}
			onOpenChange={(o) => (renameModalOpen = o)}
			renames={renamePreview}
			total={renamePreviewTotal}
			renaming={renamingFiles}
			onConfirm={confirmRename}
		/>
	</div>
</div>

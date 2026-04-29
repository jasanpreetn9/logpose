<!-- +layout -->
<script lang="ts">
	let { children } = $props();
	import { page } from '$app/state';
	import { onMount, onDestroy } from 'svelte';
	import '$lib/app.css';
	import { Library, Activity, Clock, Settings, Menu, X, Sun, Moon, Monitor } from 'lucide-svelte';
	import { Button } from '$lib/components/ui/button';
	import { ScrollArea } from '$lib/components/ui/scroll-area';
	import { Badge } from '$lib/components/ui/badge';

	import { arcs, sidebarOpen, activity, historyEvents } from '$lib/stores';
	import { navigationItems } from '$lib';
	import { cn } from '$lib/utils';
	import { api } from '$lib/api';
	import { startSSE } from '$lib/sse';
	import { theme } from '$lib/theme';

	const pathname = $derived(page.url.pathname);

	let scanningLibrary = $state(false);
	let scanningDownloads = $state(false);
	let scanError = $state<string | null>(null);

	let stopSSE: (() => void) | null = null;

	onMount(async () => {
		const [list, acts, hist] = await Promise.all([
			api.getAllEpisodes(),
			api.getActivity(),
			api.getHistory()
		]);
		arcs.set(list);
		activity.set(acts);
		historyEvents.set(hist);

		stopSSE = startSSE((ev) => {
			activity.update((l) => [ev, ...l]);
			if (ev.type === 'import') {
				historyEvents.update((l) => [ev, ...l]);
			}
		});
	});

	onDestroy(() => {
		stopSSE?.();
	});

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

	function cycleTheme() {
		theme.update((t) => (t === 'light' ? 'dark' : t === 'dark' ? 'system' : 'light'));
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

<div class="flex h-screen overflow-hidden">
	<!-- Sidebar -->
	<aside
		class={cn(
			'fixed inset-y-0 left-0 z-50 w-64 transform bg-sidebar border-r border-sidebar-border transition-transform duration-200 lg:relative lg:translate-x-0',
			$sidebarOpen ? 'translate-x-0' : '-translate-x-full'
		)}
	>
		<div class="flex h-full flex-col">
			<!-- Logo -->
			<div class="flex h-16 items-center justify-between border-b border-sidebar-border p-4">
				<div class="flex items-center gap-2">
					<div class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary">
						<Library class="h-5 w-5 text-primary-foreground" />
					</div>
					<span class="text-lg font-semibold text-sidebar-foreground">One Pace</span>
				</div>

				<Button
					variant="ghost"
					size="icon"
					class="lg:hidden"
					onclick={() => sidebarOpen.set(false)}
				>
					<X class="h-5 w-5" />
				</Button>
			</div>

			<!-- Main Nav -->
			<nav class="space-y-1 border-b border-sidebar-border p-4">
				{#each navigationItems as item}
					{@const active = pathname.startsWith(item.href)}
					<Button
						href={item.href}
						variant={active ? 'secondary' : 'ghost'}
						class={cn('w-full justify-start', active && 'bg-sidebar-accent')}
					>
						<item.icon class="mr-3 h-4 w-4" />
						{item.label}
					</Button>
				{/each}
			</nav>

			<!-- Arc List -->
			{#if pathname.startsWith('/library')}
				<div class="border-b border-sidebar-border px-4 py-3">
					<h3 class="text-sm font-semibold uppercase tracking-wider text-muted-foreground">Arcs</h3>
				</div>
				<div class="flex-1 min-h-0">
					<ScrollArea class="h-full">
						<div class="space-y-0.5 p-2">
							{#each $arcs as arc}
								<a
									href={`/library/${arc.arc}`}
									class={cn(
										'relative block w-full rounded-md px-3 py-2 text-left transition-colors hover:bg-sidebar-accent',
										pathname === `/library/${arc.arc}` && 'bg-sidebar-accent'
									)}
								>
									{#if pathname === `/library/${arc.arc}`}
										<div class="absolute left-0 top-0 h-full w-1 bg-primary rounded-r"></div>
									{/if}

									<div class="flex items-center justify-between gap-2">
										<div class="min-w-0">
											<div class="flex items-center gap-2">
												<span class="text-xs text-muted-foreground font-medium">
													#{arc.arc}
												</span>
												<span class="truncate text-sm font-medium text-sidebar-foreground">
													{arc.title}
												</span>
											</div>
										</div>

										<Badge
											variant="secondary"
											class={cn(
												'text-xs',
												arc.episodesDownloaded === arc.episodeCount && arc.episodeCount > 0
													? 'bg-green-600 text-white'
													: ''
											)}
										>
											{arc.episodesDownloaded} / {arc.episodeCount}
										</Badge>
									</div>
								</a>
							{/each}
						</div>
					</ScrollArea>
				</div>
			{/if}
		</div>
	</aside>

	<!-- Main Content -->
	<div class="flex flex-1 flex-col overflow-hidden">
		<header
			class="flex h-16 items-center justify-between border-b border-border bg-(--header) px-4 lg:px-6"
		>
			<div class="flex items-center gap-4">
				<Button variant="ghost" size="icon" class="lg:hidden" onclick={() => sidebarOpen.set(true)}>
					<Menu class="h-5 w-5" />
				</Button>
				<h1 class="text-xl font-semibold text-foreground">One Pace Library</h1>
			</div>

			<div class="flex flex-col items-end gap-1">
				<div class="flex items-center gap-3">
					<Button variant="ghost" size="icon" onclick={cycleTheme} title="Toggle theme">
						{#if $theme === 'light'}
							<Sun class="h-4 w-4" />
						{:else if $theme === 'dark'}
							<Moon class="h-4 w-4" />
						{:else}
							<Monitor class="h-4 w-4" />
						{/if}
					</Button>
					<Button
						variant="outline"
						size="sm"
						disabled={scanningDownloads}
						onclick={handleScanDownloads}
					>
						{scanningDownloads ? 'Scanning…' : 'Scan Downloads'}
					</Button>
					<Button size="sm" disabled={scanningLibrary} onclick={handleScanLibrary}>
						{scanningLibrary ? 'Scanning…' : 'Scan Library'}
					</Button>
				</div>
				{#if scanError}
					<p class="text-xs text-red-500">{scanError}</p>
				{/if}
			</div>
		</header>

		<main class="flex-1 overflow-auto p-4">
			{@render children()}
		</main>
	</div>
</div>

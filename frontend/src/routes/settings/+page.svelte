<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { Button } from '$lib/components/ui/button';

	let cfg = $state<AppConfig | null>(null);
	let qbPassword = $state('');
	let loading = $state(true);
	let saving = $state(false);
	let loadError = $state<string | null>(null);
	let saved = $state(false);
	let fieldErrors = $state<Record<string, string>>({});

	onMount(async () => {
		try {
			cfg = await api.getConfig();
		} catch (e) {
			loadError = e instanceof Error ? e.message : 'Failed to load config';
		} finally {
			loading = false;
		}
	});

	async function save() {
		if (!cfg) return;
		saving = true;
		fieldErrors = {};
		saved = false;
		const result = await api.updateConfig({ ...cfg, qbPassword: qbPassword || undefined });
		saving = false;
		if (result.errors) {
			fieldErrors = result.errors;
		} else {
			saved = true;
		}
	}
</script>

<div class="p-6 max-w-2xl space-y-8">
	<div>
		<h1 class="text-2xl font-bold">Settings</h1>
		<p class="text-muted-foreground text-sm mt-1">Configure paths, metadata, and qBittorrent</p>
	</div>

	{#if loading}
		<p class="text-muted-foreground text-sm">Loading…</p>
	{:else if cfg}
		<form onsubmit={(e) => { e.preventDefault(); save(); }} class="space-y-8">

			<!-- Paths -->
			<section class="space-y-4">
				<h2 class="text-base font-semibold border-b border-border pb-2">Paths</h2>
				<div class="space-y-1.5">
					<label class="block text-sm font-medium">Library Path</label>
					<input type="text" bind:value={cfg.libraryPath} class="field" />
					{#if fieldErrors.libraryPath}<p class="text-xs text-red-500">{fieldErrors.libraryPath}</p>{/if}
				</div>
				<div class="space-y-1.5">
					<label class="block text-sm font-medium">Downloads Path</label>
					<input type="text" bind:value={cfg.downloadPath} class="field" />
					{#if fieldErrors.downloadPath}<p class="text-xs text-red-500">{fieldErrors.downloadPath}</p>{/if}
				</div>
				<div class="space-y-1.5">
					<label class="block text-sm font-medium">Library JSON Path</label>
					<input type="text" bind:value={cfg.libraryJsonPath} class="field" />
					{#if fieldErrors.libraryJsonPath}<p class="text-xs text-red-500">{fieldErrors.libraryJsonPath}</p>{/if}
				</div>
			</section>

			<!-- Metadata -->
			<section class="space-y-4">
				<h2 class="text-base font-semibold border-b border-border pb-2">Metadata</h2>
				<div class="space-y-1.5">
					<label class="block text-sm font-medium">Episodes URL</label>
					<input type="text" bind:value={cfg.metadataEpisodesUrl} class="field" />
					{#if fieldErrors.metadataEpisodesUrl}<p class="text-xs text-red-500">{fieldErrors.metadataEpisodesUrl}</p>{/if}
				</div>
				<div class="space-y-1.5">
					<label class="block text-sm font-medium">Arcs URL</label>
					<input type="text" bind:value={cfg.metadataArcsUrl} class="field" />
					{#if fieldErrors.metadataArcsUrl}<p class="text-xs text-red-500">{fieldErrors.metadataArcsUrl}</p>{/if}
				</div>
				<div class="space-y-1.5">
					<label class="block text-sm font-medium">Refresh Interval</label>
					<input type="text" bind:value={cfg.metadataRefreshInterval} placeholder="e.g. 24h, 30m" class="field" />
					{#if fieldErrors.metadataRefreshInterval}
						<p class="text-xs text-red-500">{fieldErrors.metadataRefreshInterval}</p>
					{/if}
				</div>
			</section>

			<!-- qBittorrent -->
			<section class="space-y-4">
				<h2 class="text-base font-semibold border-b border-border pb-2">qBittorrent</h2>
				<label class="flex items-center gap-3 text-sm">
					<input type="checkbox" bind:checked={cfg.qbEnabled} class="h-4 w-4" />
					Enabled
				</label>
				<div class="space-y-1.5">
					<label class="block text-sm font-medium">Host</label>
					<input type="text" bind:value={cfg.qbHost} placeholder="http://127.0.0.1:8080" class="field" />
					{#if fieldErrors.qbHost}<p class="text-xs text-red-500">{fieldErrors.qbHost}</p>{/if}
				</div>
				<div class="space-y-1.5">
					<label class="block text-sm font-medium">Username</label>
					<input type="text" bind:value={cfg.qbUsername} class="field" />
				</div>
				<div class="space-y-1.5">
					<label class="block text-sm font-medium">Password</label>
					<input type="password" bind:value={qbPassword} placeholder="Leave blank to keep current" class="field" />
				</div>
			</section>

			<!-- Server -->
			<section class="space-y-4">
				<h2 class="text-base font-semibold border-b border-border pb-2">Server</h2>
				<div class="space-y-1.5">
					<label class="block text-sm font-medium">Port</label>
					<input type="text" bind:value={cfg.port} class="field" />
					{#if fieldErrors.port}<p class="text-xs text-red-500">{fieldErrors.port}</p>{/if}
				</div>
			</section>

			<div class="flex items-center gap-4">
				<Button type="submit" disabled={saving}>
					{saving ? 'Saving…' : 'Save Settings'}
				</Button>
				{#if saved}
					<p class="text-sm text-green-500">Settings saved.</p>
				{/if}
			</div>
		</form>
	{:else if loadError}
		<p class="text-sm text-red-500">{loadError}</p>
	{/if}
</div>

<style>
	.field {
		width: 100%;
		border-radius: 0.375rem;
		border: 1px solid var(--border);
		background: var(--background);
		padding: 0.5rem 0.75rem;
		font-size: 0.875rem;
		color: var(--foreground);
		outline: none;
	}
	.field:focus {
		box-shadow: 0 0 0 2px var(--ring);
	}
</style>

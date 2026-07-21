<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { Button } from '$lib/components/ui/button';
	import { RefreshCw, Plug } from 'lucide-svelte';

	let cfg = $state<AppConfig | null>(null);
	let qbPassword = $state('');
	let jellyfinApiKey = $state('');
	let loading = $state(true);
	let saving = $state(false);
	let loadError = $state<string | null>(null);
	let saved = $state(false);
	let fieldErrors = $state<Record<string, string>>({});
	let saveError = $state<string | null>(null);
	let refreshingMeta = $state(false);
	let refreshResult = $state<string | null>(null);
	let refreshError = $state<string | null>(null);
	let testingQb = $state(false);
	let qbTestResult = $state<string | null>(null);
	let qbTestError = $state<string | null>(null);

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
		saveError = null;
		try {
			const result = await api.updateConfig({
				...cfg,
				qbPassword: qbPassword || undefined,
				jellyfinApiKey: jellyfinApiKey || undefined
			});
			if (result.errors) {
				fieldErrors = result.errors;
			} else if (result.error) {
				saveError = result.error;
			} else {
				saved = true;
				qbPassword = '';
				jellyfinApiKey = '';
			}
		} catch (e) {
			saveError = e instanceof Error ? e.message : 'Failed to save settings';
		} finally {
			saving = false;
		}
	}

	async function testQbit() {
		if (!cfg) return;
		testingQb = true;
		qbTestResult = null;
		qbTestError = null;
		try {
			const res = await api.testQBittorrent({
				host: cfg.qbHost,
				username: cfg.qbUsername,
				password: qbPassword || undefined
			});
			if (res.ok) {
				qbTestResult = res.version ? `Connected — qBittorrent v${res.version}` : 'Connected';
			} else {
				qbTestError = res.error ?? 'Connection failed';
			}
		} catch (e) {
			qbTestError = e instanceof Error ? e.message : 'Connection failed';
		} finally {
			testingQb = false;
		}
	}

	async function refreshMetadata() {
		refreshingMeta = true;
		refreshResult = null;
		refreshError = null;
		try {
			const res = await api.refreshMetadata();
			refreshResult = `Refreshed — ${res.episodes} episodes, ${res.arcs} arcs, ${res.nfosUpdated} NFOs regenerated${res.grabbed > 0 ? `, ${res.grabbed} auto-grabbed` : ''}.`;
		} catch (e) {
			refreshError = e instanceof Error ? e.message : 'Metadata refresh failed';
		} finally {
			refreshingMeta = false;
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
				<div class="flex items-center gap-4 pt-1">
					<Button type="button" variant="outline" size="sm" onclick={refreshMetadata} disabled={refreshingMeta}>
						<RefreshCw class="mr-2 h-4 w-4 {refreshingMeta ? 'animate-spin' : ''}" />
						{refreshingMeta ? 'Refreshing…' : 'Refresh Now'}
					</Button>
					{#if refreshResult}
						<p class="text-sm text-green-500">{refreshResult}</p>
					{:else if refreshError}
						<p class="text-sm text-red-500">{refreshError}</p>
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
				<div class="flex items-center gap-4 pt-1">
					<Button type="button" variant="outline" size="sm" onclick={testQbit} disabled={testingQb}>
						<Plug class="mr-2 h-4 w-4" />
						{testingQb ? 'Testing…' : 'Test Connection'}
					</Button>
					{#if qbTestResult}
						<p class="text-sm text-green-500">{qbTestResult}</p>
					{:else if qbTestError}
						<p class="text-sm text-red-500">{qbTestError}</p>
					{/if}
				</div>
			</section>

			<!-- Automation -->
			<section class="space-y-4">
				<h2 class="text-base font-semibold border-b border-border pb-2">Automation</h2>
				<label class="flex items-center gap-3 text-sm">
					<input type="checkbox" bind:checked={cfg.autoDownload} class="h-4 w-4" />
					Auto Download monitored episodes
				</label>
				<p class="text-xs text-muted-foreground">
					Automatically queue monitored missing episodes in qBittorrent after every metadata
					refresh.
				</p>
			</section>

			<!-- Notifications -->
			<section class="space-y-4">
				<h2 class="text-base font-semibold border-b border-border pb-2">Notifications</h2>
				<div class="space-y-1.5">
					<label class="block text-sm font-medium">Discord Webhook URL</label>
					<input
						type="text"
						bind:value={cfg.discordWebhookUrl}
						placeholder="https://discord.com/api/webhooks/..."
						class="field"
					/>
					{#if fieldErrors.discordWebhookUrl}
						<p class="text-xs text-red-500">{fieldErrors.discordWebhookUrl}</p>
					{/if}
				</div>
				<div class="space-y-1.5">
					<label class="block text-sm font-medium">Jellyfin URL</label>
					<input
						type="text"
						bind:value={cfg.jellyfinUrl}
						placeholder="http://127.0.0.1:8096"
						class="field"
					/>
					{#if fieldErrors.jellyfinUrl}<p class="text-xs text-red-500">{fieldErrors.jellyfinUrl}</p>{/if}
				</div>
				<div class="space-y-1.5">
					<label class="block text-sm font-medium">Jellyfin API Key</label>
					<input
						type="password"
						bind:value={jellyfinApiKey}
						placeholder="Leave blank to keep current"
						class="field"
					/>
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
				{:else if saveError}
					<p class="text-sm text-red-500">{saveError}</p>
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

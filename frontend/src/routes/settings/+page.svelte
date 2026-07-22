<script lang="ts">
	import { onMount } from 'svelte';
	import { api } from '$lib/api';
	import { Switch } from '$lib/components/ui/switch';

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

	const section = 'rounded-md border border-border bg-card p-[18px]';
	const sectionTitle = 'mb-3.5 text-[13px] font-bold text-card-foreground';
	const fieldLabel = 'text-[11px] text-muted-foreground';
	const field =
		'mt-1.5 block w-full box-border rounded border border-[#2a2f3a] bg-background px-2.5 py-2 text-[12px] text-card-foreground outline-none focus:ring-2 focus:ring-ring';
	const fieldMono = field + ' font-mono';
	const accentButton =
		'cursor-pointer rounded-[5px] px-3.5 py-2 text-[12px] font-semibold text-primary transition-colors hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-60';
</script>

{#if loading}
	<p class="text-sm text-muted-foreground">Loading…</p>
{:else if cfg}
	<form
		onsubmit={(e) => {
			e.preventDefault();
			save();
		}}
		class="flex max-w-[560px] flex-col gap-4"
	>
		<!-- Paths -->
		<section class={section}>
			<div class={sectionTitle}>Paths</div>
			<div class="flex flex-col gap-2.5">
				<label class={fieldLabel}>
					Library Path
					<input type="text" bind:value={cfg.libraryPath} class={fieldMono} />
					{#if fieldErrors.libraryPath}<p class="mt-1 text-[10.5px] text-destructive">{fieldErrors.libraryPath}</p>{/if}
				</label>
				<label class={fieldLabel}>
					Download Path
					<input type="text" bind:value={cfg.downloadPath} class={fieldMono} />
					{#if fieldErrors.downloadPath}<p class="mt-1 text-[10.5px] text-destructive">{fieldErrors.downloadPath}</p>{/if}
				</label>
				<label class={fieldLabel}>
					Library JSON Path
					<input type="text" bind:value={cfg.libraryJsonPath} class={fieldMono} />
					{#if fieldErrors.libraryJsonPath}<p class="mt-1 text-[10.5px] text-destructive">{fieldErrors.libraryJsonPath}</p>{/if}
				</label>
			</div>
		</section>

		<!-- Metadata -->
		<section class={section}>
			<div class={sectionTitle}>Metadata</div>
			<div class="flex flex-col gap-2.5">
				<label class={fieldLabel}>
					Episodes URL
					<input type="text" bind:value={cfg.metadataEpisodesUrl} class={fieldMono} />
					{#if fieldErrors.metadataEpisodesUrl}<p class="mt-1 text-[10.5px] text-destructive">{fieldErrors.metadataEpisodesUrl}</p>{/if}
				</label>
				<label class={fieldLabel}>
					Arcs URL
					<input type="text" bind:value={cfg.metadataArcsUrl} class={fieldMono} />
					{#if fieldErrors.metadataArcsUrl}<p class="mt-1 text-[10.5px] text-destructive">{fieldErrors.metadataArcsUrl}</p>{/if}
				</label>
				<label class={fieldLabel}>
					Refresh Interval
					<input type="text" bind:value={cfg.metadataRefreshInterval} placeholder="e.g. 24h, 30m" class={fieldMono} />
					{#if fieldErrors.metadataRefreshInterval}
						<p class="mt-1 text-[10.5px] text-destructive">{fieldErrors.metadataRefreshInterval}</p>
					{/if}
				</label>
			</div>
			<div class="mt-3.5 flex items-center gap-3">
				<button type="button" class={accentButton} style="background:#233042" disabled={refreshingMeta} onclick={refreshMetadata}>
					{refreshingMeta ? 'Refreshing…' : 'Refresh Now'}
				</button>
				{#if refreshResult}
					<span class="text-[11.5px]" style="color:#3ecf8e">{refreshResult}</span>
				{:else if refreshError}
					<span class="text-[11.5px] text-destructive">{refreshError}</span>
				{/if}
			</div>
		</section>

		<!-- qBittorrent -->
		<section class={section}>
			<div class="mb-3.5 flex items-center justify-between">
				<div class="text-[13px] font-bold text-card-foreground">qBittorrent</div>
				<Switch checked={cfg.qbEnabled} onCheckedChange={(v) => cfg && (cfg.qbEnabled = v)} />
			</div>
			<div class="flex flex-col gap-2.5">
				<label class={fieldLabel}>
					Host
					<input type="text" bind:value={cfg.qbHost} placeholder="http://127.0.0.1:8080/" class={fieldMono} />
					{#if fieldErrors.qbHost}<p class="mt-1 text-[10.5px] text-destructive">{fieldErrors.qbHost}</p>{/if}
				</label>
				<label class={fieldLabel}>
					Username
					<input type="text" bind:value={cfg.qbUsername} class={field} />
				</label>
				<label class={fieldLabel}>
					Password
					<input type="password" bind:value={qbPassword} placeholder="leave blank to keep current" class={field} />
				</label>
			</div>
			<div class="mt-3.5 flex items-center gap-3">
				<button type="button" class={accentButton} style="background:#233042" disabled={testingQb} onclick={testQbit}>
					{testingQb ? 'Testing…' : 'Test Connection'}
				</button>
				{#if qbTestResult}
					<span class="text-[11px]" style="color:#3ecf8e">{qbTestResult}</span>
				{:else if qbTestError}
					<span class="text-[11px] text-destructive">{qbTestError}</span>
				{/if}
			</div>
		</section>

		<!-- Automation -->
		<section class={section}>
			<div class={sectionTitle}>Automation</div>
			<div class="flex items-center justify-between">
				<div>
					<div class="text-[12.5px] text-foreground">Auto-Download</div>
					<div class="mt-0.5 text-[11px] text-muted-foreground">
						Auto-queue monitored missing episodes after every metadata refresh
					</div>
				</div>
				<Switch checked={cfg.autoDownload} onCheckedChange={(v) => cfg && (cfg.autoDownload = v)} />
			</div>
		</section>

		<!-- Notifications -->
		<section class={section}>
			<div class={sectionTitle}>Notifications</div>
			<div class="flex flex-col gap-2.5">
				<label class={fieldLabel}>
					Discord Webhook URL
					<input
						type="text"
						bind:value={cfg.discordWebhookUrl}
						placeholder="blank disables Discord notifications"
						class={fieldMono}
					/>
					{#if fieldErrors.discordWebhookUrl}<p class="mt-1 text-[10.5px] text-destructive">{fieldErrors.discordWebhookUrl}</p>{/if}
				</label>
				<label class={fieldLabel}>
					Jellyfin URL
					<input
						type="text"
						bind:value={cfg.jellyfinUrl}
						placeholder="blank disables Jellyfin refresh"
						class={fieldMono}
					/>
					{#if fieldErrors.jellyfinUrl}<p class="mt-1 text-[10.5px] text-destructive">{fieldErrors.jellyfinUrl}</p>{/if}
				</label>
				<label class={fieldLabel}>
					Jellyfin API Key
					<input type="password" bind:value={jellyfinApiKey} placeholder="leave blank to keep current" class={field} />
				</label>
			</div>
		</section>

		<!-- Server -->
		<section class={section}>
			<div class={sectionTitle}>Server</div>
			<label class={fieldLabel}>
				Port
				<input type="text" bind:value={cfg.port} class={fieldMono} style="max-width:140px" />
				{#if fieldErrors.port}<p class="mt-1 text-[10.5px] text-destructive">{fieldErrors.port}</p>{/if}
			</label>
		</section>

		<div class="flex items-center gap-3">
			<button type="submit" class={accentButton + ' font-bold'} style="background:#233042" disabled={saving}>
				{saving ? 'Saving…' : 'Save Settings'}
			</button>
			{#if saved}
				<span class="text-[11.5px]" style="color:#3ecf8e">Saved</span>
			{:else if saveError}
				<span class="text-[11.5px] text-destructive">{saveError}</span>
			{/if}
		</div>
	</form>
{:else if loadError}
	<p class="text-sm text-destructive">{loadError}</p>
{/if}

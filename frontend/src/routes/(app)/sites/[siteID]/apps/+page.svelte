<script lang="ts">
	import { onMount } from 'svelte';
	import * as appsApi from '$lib/api/apps';
	import { ApiError } from '$lib/api/client';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Textarea from '$lib/components/ui/Textarea.svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import { confirm } from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { Boxes, Plug } from 'lucide-svelte';
	import type { Site } from '$lib/api/types';

	let { data }: { data: { site: Site } } = $props();
	const siteID = $derived(data.site.id);

	let marketplace = $state<appsApi.App[]>([]);
	let installs = $state<appsApi.SiteInstall[]>([]);
	let loading = $state(true);

	let filter = $state<'all' | string>('all');

	// Install dialog state.
	let dialogOpen = $state(false);
	let dialogApp = $state<appsApi.App | null>(null);
	let dialogCreds = $state<Record<string, string>>({});
	let dialogNotes = $state('');
	let dialogSaving = $state(false);

	async function load() {
		loading = true;
		try {
			[marketplace, installs] = await Promise.all([
				appsApi.listMarketplace(),
				appsApi.listSiteInstalls(siteID)
			]);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to load apps.');
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		void load();
	});

	const installedByID = $derived<Record<string, appsApi.SiteInstall>>(
		Object.fromEntries(installs.map((i) => [i.app_id, i]))
	);

	const categories = $derived(
		Array.from(new Set(marketplace.map((a) => a.category))).sort()
	);

	const visibleApps = $derived(
		filter === 'all' ? marketplace : marketplace.filter((a) => a.category === filter)
	);

	function openInstallDialog(app: appsApi.App) {
		dialogApp = app;
		dialogCreds = Object.fromEntries(app.credential_fields.map((f) => [f.key, '']));
		dialogNotes = '';
		dialogOpen = true;
	}

	async function submitInstall() {
		if (!dialogApp || dialogSaving) return;
		// Required check matches the backend so the dialog surfaces the
		// error inline before submitting.
		for (const f of dialogApp.credential_fields) {
			if (f.required && !dialogCreds[f.key]?.trim()) {
				toast.error(`Required field missing: ${f.label}`);
				return;
			}
		}
		dialogSaving = true;
		try {
			await appsApi.install(siteID, dialogApp.id, {
				credentials: dialogCreds,
				notes: dialogNotes.trim()
			});
			dialogOpen = false;
			toast.success(`${dialogApp.name} installed.`);
			await load();
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Install failed.';
			toast.error(msg);
		} finally {
			dialogSaving = false;
		}
	}

	async function revokeInstall(app: appsApi.App) {
		const ok = await confirm({
			title: `Revoke ${app.name}?`,
			message: `Stored credentials for ${app.name} will be removed. The agent will no longer be able to use this integration. You can re-install at any time with new credentials.`,
			confirmText: 'Revoke',
			cancelText: 'Cancel',
			variant: 'danger'
		});
		if (!ok) return;
		try {
			await appsApi.revoke(siteID, app.id);
			toast.success(`${app.name} revoked.`);
			await load();
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Revoke failed.';
			toast.error(msg);
		}
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">Apps</h1>
		<p class="text-[13px] text-text-secondary">
			Install third-party integrations once; any AI agent connected via MCP can use the stored
			credentials. Slice A: marketplace + install ledger. Slice B will add the upstream MCP
			proxy so the agent can dial each installed app's tools under a namespaced surface (e.g.
			<code class="font-mono text-[11.5px]">stripe.create_payment_link</code>).
		</p>
	</header>

	{#if loading}
		<div class="mt-8 grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
			<Skeleton class="h-32" />
			<Skeleton class="h-32" />
			<Skeleton class="h-32" />
		</div>
	{:else}
		<section class="mt-6">
			<header class="flex items-center justify-between gap-3">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Installed ({installs.length})
				</h2>
			</header>
			{#if installs.length === 0}
				<div class="mt-3">
					<EmptyState
						title="No apps installed yet"
						description="Browse the marketplace below and click Install on any app you want to use."
					>
						{#snippet icon()}
							<Plug class="h-6 w-6" />
						{/snippet}
					</EmptyState>
				</div>
			{:else}
				<ul class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
					{#each installs as inst (inst.app_id)}
						<li>
							<Card padding="md">
								<div class="flex items-start gap-3">
									{#if inst.app.icon_url}
										<img
											src={inst.app.icon_url}
											alt=""
											class="h-9 w-9 rounded-lg border border-border-light bg-bg-elevated object-contain"
										/>
									{:else}
										<span
											class="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-border-light bg-bg-elevated text-text-secondary"
										>
											<Boxes class="h-4 w-4" strokeWidth={1.5} />
										</span>
									{/if}
									<div class="min-w-0 flex-1">
										<div class="flex items-center justify-between gap-2">
											<p class="text-[13px] font-medium text-text-primary">{inst.app.name}</p>
											<Badge variant={inst.status === 'active' ? 'success' : 'warning'}>
												{inst.status}
											</Badge>
										</div>
										<p class="text-[11px] text-text-muted font-mono uppercase tracking-[0.16em]">
											{inst.app.category}
										</p>
										<p class="mt-2 text-[12px] text-text-muted">
											Credentials set: {inst.credentials_set.length === 0
												? 'none'
												: inst.credentials_set.join(', ')}
										</p>
									</div>
								</div>
								<div class="mt-3 flex gap-2">
									<Button
										variant="secondary"
										onclick={() => openInstallDialog(inst.app)}
									>
										Reconfigure
									</Button>
									<Button variant="secondary" onclick={() => revokeInstall(inst.app)}>
										Revoke
									</Button>
								</div>
							</Card>
						</li>
					{/each}
				</ul>
			{/if}
		</section>

		<section class="mt-10">
			<header class="flex items-center justify-between gap-3">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Marketplace ({marketplace.length})
				</h2>
				<div class="flex gap-1.5 overflow-x-auto">
					<button
						type="button"
						class="rounded-full border px-2.5 py-1 text-[11px] font-medium transition-colors {filter === 'all'
							? 'border-accent bg-accent/10 text-text-primary'
							: 'border-border-light text-text-muted hover:text-text-primary'}"
						onclick={() => (filter = 'all')}
					>
						All
					</button>
					{#each categories as cat (cat)}
						<button
							type="button"
							class="rounded-full border px-2.5 py-1 text-[11px] font-medium transition-colors {filter === cat
								? 'border-accent bg-accent/10 text-text-primary'
								: 'border-border-light text-text-muted hover:text-text-primary'}"
							onclick={() => (filter = cat)}
						>
							{cat}
						</button>
					{/each}
				</div>
			</header>
			<ul class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
				{#each visibleApps as app (app.id)}
					{@const installed = !!installedByID[app.id]}
					<li>
						<Card padding="md">
							<div class="flex items-start gap-3">
								{#if app.icon_url}
									<img
										src={app.icon_url}
										alt=""
										class="h-9 w-9 rounded-lg border border-border-light bg-bg-elevated object-contain"
									/>
								{:else}
									<span
										class="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-border-light bg-bg-elevated text-text-secondary"
									>
										<Boxes class="h-4 w-4" strokeWidth={1.5} />
									</span>
								{/if}
								<div class="min-w-0 flex-1">
									<div class="flex items-center justify-between gap-2">
										<p class="text-[13px] font-medium text-text-primary">{app.name}</p>
										{#if app.is_curated}
											<Badge variant="info">Curated</Badge>
										{/if}
									</div>
									<p class="text-[11px] text-text-muted font-mono uppercase tracking-[0.16em]">
										{app.category}
									</p>
								</div>
							</div>
							<p class="mt-3 text-[12px] text-text-secondary">{app.description}</p>
							<div class="mt-3 flex gap-2">
								{#if installed}
									<Button variant="secondary" disabled>Installed</Button>
								{:else}
									<Button variant="primary" onclick={() => openInstallDialog(app)}>
										Install
									</Button>
								{/if}
								{#if app.docs_url}
									<a
										href={app.docs_url}
										target="_blank"
										rel="noopener noreferrer"
										class="inline-flex items-center rounded-lg border border-border-light px-3 py-1.5 text-[12px] text-text-secondary hover:text-text-primary"
									>
										Docs
									</a>
								{/if}
							</div>
						</Card>
					</li>
				{/each}
			</ul>
		</section>
	{/if}
</div>

<Dialog
	bind:open={dialogOpen}
	size="md"
	title={dialogApp ? `Install ${dialogApp.name}` : 'Install'}
	description={dialogApp?.description ?? ''}
>
	{#if dialogApp}
		<div class="flex flex-col gap-4">
			{#if dialogApp.credential_fields.length === 0}
				<p class="text-[12px] text-text-muted">
					This app has no credential fields to configure.
				</p>
			{/if}
			{#each dialogApp.credential_fields as f (f.key)}
				<Input
					label={`${f.label}${f.required ? ' *' : ''}`}
					type={f.kind === 'secret' ? 'password' : 'text'}
					placeholder={f.placeholder ?? ''}
					hint={f.help ?? ''}
					autocomplete="off"
					spellcheck={false}
					value={dialogCreds[f.key] ?? ''}
					oninput={(e) => {
						dialogCreds = {
							...dialogCreds,
							[f.key]: (e.currentTarget as HTMLInputElement).value
						};
					}}
				/>
			{/each}
			<Textarea
				label="Notes (optional)"
				rows={3}
				placeholder="What this install is for, who set it up, etc."
				value={dialogNotes}
				oninput={(e) =>
					(dialogNotes = (e.currentTarget as HTMLTextAreaElement).value)}
			/>
			{#if dialogApp.docs_url}
				<p class="text-[11px] text-text-muted">
					<a
						href={dialogApp.docs_url}
						target="_blank"
						rel="noopener noreferrer"
						class="text-accent underline-offset-2 hover:underline"
					>
						Open {dialogApp.name} docs
					</a>
					to find these credentials.
				</p>
			{/if}
		</div>
	{/if}
	{#snippet footer()}
		<div class="flex justify-end gap-2">
			<Button variant="secondary" onclick={() => (dialogOpen = false)} disabled={dialogSaving}>
				Cancel
			</Button>
			<Button variant="primary" onclick={submitInstall} disabled={dialogSaving}>
				{dialogSaving ? 'Saving...' : 'Install'}
			</Button>
		</div>
	{/snippet}
</Dialog>

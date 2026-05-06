<script lang="ts">
	import { onDestroy } from 'svelte';
	import * as domainsApi from '$lib/api/domains';
	import type { SiteDomain } from '$lib/api/domains';
	import { ApiError } from '$lib/api/client';
	import Input from '$lib/components/ui/Input.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import { confirm } from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import type { Site } from '$lib/api/types';
	import { Plus, Trash2, RefreshCw, Star, Globe, Copy } from 'lucide-svelte';

	let { data }: { data: { site: Site } } = $props();
	const siteID = $derived(data.site.id);

	let loading = $state(true);
	let domains = $state<SiteDomain[]>([]);
	let loadError = $state<string | null>(null);

	let dialogOpen = $state(false);
	let formHostname = $state('');
	let formCanonical = $state(false);
	let formError = $state<string | null>(null);
	let submitting = $state(false);

	// Strict hostname re. Lowercase letters, digits, hyphens; at least
	// one dot; TLD >= 2 chars; no leading dot, no double-dots.
	const HOSTNAME_RE = /^(?!-)([a-z0-9-]{1,63}(?<!-)\.)+[a-z]{2,}$/;

	// Auto-refresh while any row is mid-pipeline (anything not 'live'
	// and not 'error') so editors see status flips within seconds. We
	// stop polling once everything is settled to avoid wasting cycles.
	let pollTimer: ReturnType<typeof setInterval> | null = null;

	async function load() {
		loading = true;
		loadError = null;
		try {
			const res = await domainsApi.list(siteID);
			domains = res.domains ?? [];
			schedulePolling();
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load domains.';
		} finally {
			loading = false;
		}
	}

	function schedulePolling() {
		const inFlight = domains.some(
			(d) => d.status !== 'live' && d.status !== 'error'
		);
		if (inFlight && !pollTimer) {
			pollTimer = setInterval(async () => {
				try {
					const res = await domainsApi.list(siteID);
					domains = res.domains ?? [];
					if (!domains.some((d) => d.status !== 'live' && d.status !== 'error')) {
						stopPolling();
					}
				} catch {
					// transient; keep polling
				}
			}, 4000);
		} else if (!inFlight) {
			stopPolling();
		}
	}

	function stopPolling() {
		if (pollTimer) {
			clearInterval(pollTimer);
			pollTimer = null;
		}
	}

	onDestroy(stopPolling);

	$effect(() => {
		void load();
	});

	function openAddDialog() {
		formHostname = '';
		formCanonical = domains.length === 0; // first domain auto-canonical
		formError = null;
		dialogOpen = true;
	}

	function normaliseHostname(s: string): string {
		return s.trim().toLowerCase().replace(/^https?:\/\//, '').replace(/\/.*$/, '');
	}

	async function submitAdd() {
		if (submitting) return;
		formError = null;
		const host = normaliseHostname(formHostname);
		if (!host) {
			formError = 'Hostname is required.';
			return;
		}
		if (!HOSTNAME_RE.test(host)) {
			formError = 'Enter a bare hostname like example.com or shop.example.com (no scheme, no path).';
			return;
		}
		submitting = true;
		try {
			await domainsApi.create(siteID, host, formCanonical);
			toast.success(`Added ${host}. Point DNS at the edge IP and we'll verify it.`);
			dialogOpen = false;
			await load();
		} catch (err) {
			formError = err instanceof ApiError ? err.message : 'Failed to add domain.';
		} finally {
			submitting = false;
		}
	}

	async function refresh(d: SiteDomain) {
		try {
			await domainsApi.refresh(siteID, d.id);
			toast.success(`Re-checking ${d.hostname}...`);
			// Quick local UI hint that something is happening; the
			// poller will pick up the real flip in a few seconds.
			schedulePolling();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Refresh failed.');
		}
	}

	async function setCanonical(d: SiteDomain) {
		try {
			await domainsApi.setCanonical(siteID, d.id);
			toast.success(`${d.hostname} is now the canonical URL.`);
			await load();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Failed to set canonical.');
		}
	}

	async function remove(d: SiteDomain) {
		const ok = await confirm({
			title: `Remove ${d.hostname}?`,
			message: `Visitors hitting ${d.hostname} will stop reaching this site. The certificate stays on disk but the nginx vhost is removed.`,
			confirmText: 'Remove domain',
			variant: 'danger'
		});
		if (!ok) return;
		try {
			await domainsApi.remove(siteID, d.id);
			toast.success(`Removed ${d.hostname}.`);
			await load();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Delete failed.');
		}
	}

	function statusVariant(s: SiteDomain['status']): { label: string; cls: string } {
		switch (s) {
			case 'pending':
				return { label: 'Awaiting DNS', cls: 'bg-warning/10 text-warning ring-warning/20' };
			case 'verified':
				return { label: 'Issuing TLS', cls: 'bg-info/10 text-info ring-info/20' };
			case 'cert_ready':
				return { label: 'Finalising', cls: 'bg-info/10 text-info ring-info/20' };
			case 'live':
				return { label: 'Live', cls: 'bg-success/10 text-success ring-success/20' };
			case 'error':
				return { label: 'Error', cls: 'bg-danger/10 text-danger ring-danger/20' };
		}
	}

	function copy(text: string) {
		navigator.clipboard?.writeText(text).then(
			() => toast.success('Copied'),
			() => toast.error('Copy failed')
		);
	}
</script>

<div class="mx-auto max-w-5xl px-6 py-8">
	<header class="flex items-start justify-between gap-4">
		<div class="flex flex-col gap-1.5">
			<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
				Domains
			</h1>
			<p class="text-[13px] text-text-secondary">
				Custom hostnames the live site answers from. We verify DNS, issue a Let's Encrypt
				certificate, and rewrite nginx so the live site serves from the new hostname with
				the same A+ security headers. Where the built artifact ships to is separate, see
				<a href="/sites/{siteID}/settings/deployment" class="text-accent underline-offset-2 hover:underline">Deployment</a>.
			</p>
		</div>
		<Button on:click={openAddDialog}>
			<Plus size={14} strokeWidth={1.5} class="mr-1" /> Add domain
		</Button>
	</header>

	<Card padding="md" class="mt-6">
		<div class="flex items-start gap-3">
			<span
				class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border border-border-light bg-bg-elevated text-text-secondary"
			>
				<Globe size={14} strokeWidth={1.5} />
			</span>
			<div class="text-[12.5px] text-text-secondary leading-relaxed">
				<p class="text-[13px] font-medium text-text-primary">How it works</p>
				<ol class="mt-1 list-decimal pl-5 space-y-0.5">
					<li>Add the hostname here. We mint a one-time verify token.</li>
					<li>Point an <code class="font-mono text-[11px]">A</code> record at our
						edge IP. We auto-create the record on Cloudflare zones we control.</li>
					<li>We probe <code class="font-mono text-[11px]">/.well-known/atomic-verify/&lt;token&gt;</code>
						over plain HTTP. When it answers, the row flips to verified.</li>
					<li>certbot HTTP-01 issues a Let's Encrypt certificate.</li>
					<li>We render the per-domain nginx server block, reload, and the row
						flips to live.</li>
				</ol>
			</div>
		</div>
	</Card>

	{#if loading}
		<div class="mt-8 flex justify-center"><Spinner /></div>
	{:else if loadError}
		<Card padding="md" class="mt-6 border-danger/30">
			<p class="text-[13px] text-danger">{loadError}</p>
		</Card>
	{:else if domains.length === 0}
		<EmptyState
			title="No custom domains yet"
			description="Add one to make this site answer from your own URL."
			class="mt-8"
		/>
	{:else}
		<ul class="mt-6 flex flex-col gap-3">
			{#each domains as d (d.id)}
				{@const sv = statusVariant(d.status)}
				<li>
					<Card padding="md">
						<div class="flex flex-wrap items-start justify-between gap-3">
							<div class="min-w-0 flex-1">
								<div class="flex flex-wrap items-center gap-2">
									<span class="font-mono text-[14px] font-medium text-text-primary">
										{d.hostname}
									</span>
									<span class="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium ring-1 ring-inset {sv.cls}">
										{sv.label}
									</span>
									{#if d.is_canonical === 1}
										<span class="inline-flex items-center rounded-full bg-accent/10 text-accent ring-accent/20 ring-1 ring-inset px-2 py-0.5 text-[11px] font-medium">
											<Star size={10} strokeWidth={1.5} class="mr-1" /> Canonical
										</span>
									{/if}
								</div>

								{#if d.status === 'pending'}
									<p class="mt-2 text-[12px] text-text-secondary">
										Add an <code class="font-mono text-[11px]">A</code> record:
										<code class="font-mono text-[11px] text-text-primary">{d.hostname}</code>
										&nbsp;→&nbsp;
										<code class="font-mono text-[11px] text-text-primary">[your edge IP]</code>.
										DNS propagation usually takes a minute or two.
									</p>
								{:else if d.status === 'error' && d.last_error}
									<p class="mt-2 text-[12px] text-danger">
										{d.last_error}
									</p>
								{:else if d.status === 'live'}
									<p class="mt-2 text-[12px] text-text-muted">
										Last checked {d.last_check_at ? new Date(d.last_check_at).toLocaleString() : 'never'}
									</p>
								{/if}

								<details class="mt-2">
									<summary class="cursor-pointer text-[11px] text-text-muted hover:text-text-secondary">Verification token</summary>
									<div class="mt-2 flex items-center gap-2">
										<code class="block flex-1 truncate rounded-md border border-border-light bg-bg-elevated px-2 py-1 font-mono text-[11px] text-text-secondary">
											{d.verify_token}
										</code>
										<button
											type="button"
											class="inline-flex h-7 w-7 items-center justify-center rounded-md border border-border-light text-text-secondary hover:bg-bg-elevated"
											onclick={() => copy(d.verify_token)}
											aria-label="Copy verify token"
										>
											<Copy size={12} strokeWidth={1.5} />
										</button>
									</div>
									<p class="mt-1 text-[11px] text-text-muted">
										Curl test: <code class="font-mono text-[11px]">curl http://{d.hostname}/.well-known/atomic-verify/{d.verify_token}</code>
									</p>
								</details>
							</div>

							<div class="flex shrink-0 items-center gap-1">
								<button
									type="button"
									class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-border-light text-text-secondary hover:bg-bg-elevated"
									onclick={() => refresh(d)}
									title="Re-check now"
									aria-label="Refresh status"
								>
									<RefreshCw size={14} strokeWidth={1.5} />
								</button>
								{#if d.status === 'live' && d.is_canonical !== 1}
									<button
										type="button"
										class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-border-light text-text-secondary hover:bg-bg-elevated"
										onclick={() => setCanonical(d)}
										title="Make canonical"
										aria-label="Set as canonical"
									>
										<Star size={14} strokeWidth={1.5} />
									</button>
								{/if}
								<button
									type="button"
									class="inline-flex h-8 w-8 items-center justify-center rounded-md border border-border-light text-danger hover:bg-danger/5"
									onclick={() => remove(d)}
									title="Remove"
									aria-label="Remove domain"
								>
									<Trash2 size={14} strokeWidth={1.5} />
								</button>
							</div>
						</div>
					</Card>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<Dialog bind:open={dialogOpen} title="Add custom domain">
	<form
		onsubmit={(e) => {
			e.preventDefault();
			void submitAdd();
		}}
		class="flex flex-col gap-4"
	>
		<div class="flex flex-col gap-1.5">
			<label for="dom-host" class="text-[12px] font-medium text-text-secondary">
				Hostname
			</label>
			<Input
				id="dom-host"
				type="text"
				bind:value={formHostname}
				placeholder="example.com"
				autocomplete="off"
				autofocus
			/>
			<p class="text-[11px] text-text-muted">
				Bare hostname only, no scheme, no path. Both <code class="font-mono text-[11px]">example.com</code>
				and <code class="font-mono text-[11px]">shop.example.com</code> work.
			</p>
		</div>

		<label class="flex items-start gap-2 text-[12px] text-text-secondary">
			<input
				type="checkbox"
				bind:checked={formCanonical}
				class="mt-0.5 h-4 w-4 rounded border-border-light"
			/>
			<span>
				Set as canonical URL.
				<span class="block text-text-muted">
					Used in <code class="font-mono text-[11px]">canonical</code> tags,
					sitemaps, and OG URLs. Each site has exactly one canonical.
				</span>
			</span>
		</label>

		{#if formError}
			<p class="text-[12px] text-danger">{formError}</p>
		{/if}

		<div class="mt-2 flex justify-end gap-2">
			<Button type="button" variant="secondary" on:click={() => (dialogOpen = false)} disabled={submitting}>
				Cancel
			</Button>
			<Button type="submit" disabled={submitting}>
				{submitting ? 'Adding...' : 'Add domain'}
			</Button>
		</div>
	</form>
</Dialog>

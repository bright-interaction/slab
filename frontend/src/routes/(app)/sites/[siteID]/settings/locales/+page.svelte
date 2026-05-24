<script lang="ts">
	import * as localesApi from '$lib/api/locales';
	import { ApiError } from '$lib/api/client';
	import Input from '$lib/components/ui/Input.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import type { Site } from '$lib/api/types';

	let { data }: { data: { site: Site } } = $props();
	const siteID = $derived(data.site.id);

	let loading = $state(true);
	let rows = $state<localesApi.SiteLocale[]>([]);

	let newLocale = $state('');
	let savingNew = $state(false);

	let pendingDelete = $state<string | null>(null);

	async function load() {
		loading = true;
		try {
			rows = await localesApi.listSiteLocales(siteID);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to load locales.');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	const hasDefault = $derived(rows.some((r) => r.is_default));

	async function addLocale() {
		const code = newLocale.trim().toLowerCase();
		if (!/^[a-z]{2}(-[a-z]{2})?$/.test(code)) {
			toast.error('Locale must be a 2-letter code (sv) or language-region pair (sv-se).');
			return;
		}
		if (rows.find((r) => r.locale === code)) {
			toast.error(`${code} is already on the list.`);
			return;
		}
		savingNew = true;
		try {
			const next = rows.length;
			await localesApi.upsertSiteLocale(siteID, {
				locale: code,
				is_default: !hasDefault,
				sort_order: next
			});
			newLocale = '';
			await load();
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Failed to add locale.';
			toast.error(msg);
		} finally {
			savingNew = false;
		}
	}

	async function setDefault(locale: string) {
		try {
			const row = rows.find((r) => r.locale === locale);
			await localesApi.upsertSiteLocale(siteID, {
				locale,
				is_default: true,
				sort_order: row?.sort_order ?? 0
			});
			await load();
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Failed to set default locale.';
			toast.error(msg);
		}
	}

	async function moveUp(locale: string) {
		const idx = rows.findIndex((r) => r.locale === locale);
		if (idx <= 0) return;
		const above = rows[idx - 1];
		const me = rows[idx];
		if (!above || !me) return;
		try {
			await localesApi.upsertSiteLocale(siteID, {
				locale: me.locale,
				is_default: me.is_default,
				sort_order: above.sort_order
			});
			await localesApi.upsertSiteLocale(siteID, {
				locale: above.locale,
				is_default: above.is_default,
				sort_order: me.sort_order
			});
			await load();
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Failed to reorder.';
			toast.error(msg);
		}
	}

	async function confirmDelete() {
		if (!pendingDelete) return;
		try {
			await localesApi.deleteSiteLocale(siteID, pendingDelete);
			toast.success(`${pendingDelete} removed.`);
			await load();
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Failed to remove locale.';
			toast.error(msg);
		} finally {
			pendingDelete = null;
		}
	}
</script>

<div class="mx-auto max-w-3xl px-6 py-8">
	<header class="flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">Locales</h1>
		<p class="text-[13px] text-text-secondary">
			One row per language your site ships in. The default locale renders at the base URL
			(<code class="font-mono text-[11.5px]">/about</code>); additional locales render under their
			prefix (<code class="font-mono text-[11.5px]">/sv/about</code>) and pick up per-page +
			per-block translations authored from the page editor.
		</p>
	</header>

	{#if loading}
		<div class="mt-8 flex items-center justify-center py-12">
			<Spinner />
		</div>
	{:else}
		<div class="mt-8 flex flex-col gap-5">
			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Add a locale
				</h2>
				<p class="mt-1 text-[12px] text-text-muted">
					Two-letter ISO 639 (<code class="font-mono text-[11.5px]">sv</code>,
					<code class="font-mono text-[11.5px]">en</code>) or language-region
					(<code class="font-mono text-[11.5px]">sv-se</code>,
					<code class="font-mono text-[11.5px]">en-gb</code>). The first locale you add becomes
					the default automatically; flip the default later with the star button.
				</p>
				<div class="mt-4 flex gap-2">
					<Input
						placeholder="sv"
						maxlength={5}
						autocomplete="off"
						spellcheck={false}
						bind:value={newLocale}
					/>
					<Button variant="primary" disabled={savingNew || !newLocale} onclick={addLocale}>
						{savingNew ? 'Adding...' : 'Add'}
					</Button>
				</div>
			</Card>

			<Card padding="none">
				<div class="border-b border-border-light px-4 py-3">
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Configured locales ({rows.length})
					</h2>
				</div>
				{#if rows.length === 0}
					<p class="px-4 py-6 text-center text-[13px] text-text-muted">
						No locales yet. Add one above to start translating pages.
					</p>
				{:else}
					<ul class="divide-y divide-border-light">
						{#each rows as row, idx (row.locale)}
							<li class="flex items-center justify-between gap-3 px-4 py-3">
								<div class="flex items-center gap-3">
									<span class="font-mono text-[13px] text-text-primary">{row.locale}</span>
									{#if row.is_default}
										<Badge variant="success">Default</Badge>
									{/if}
								</div>
								<div class="flex items-center gap-1.5">
									<button
										type="button"
										class="rounded p-1 text-text-muted hover:bg-bg-elevated hover:text-text-primary disabled:opacity-30 disabled:hover:bg-transparent"
										title="Move up"
										disabled={idx === 0}
										onclick={() => moveUp(row.locale)}
									>
										&uarr;
									</button>
									{#if !row.is_default}
										<button
											type="button"
											class="rounded p-1 text-text-muted hover:bg-bg-elevated hover:text-text-primary"
											title="Make default"
											onclick={() => setDefault(row.locale)}
										>
											&#9733;
										</button>
										<button
											type="button"
											class="rounded p-1 text-text-muted hover:bg-danger/10 hover:text-danger"
											title="Remove"
											onclick={() => (pendingDelete = row.locale)}
										>
											&times;
										</button>
									{:else}
										<span class="px-1 text-[11px] text-text-muted">locked</span>
									{/if}
								</div>
							</li>
						{/each}
					</ul>
				{/if}
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					What happens next
				</h2>
				<ol class="mt-3 list-decimal space-y-1.5 pl-5 text-[13px] text-text-secondary">
					<li>
						Open any page in the editor. A locale tabs strip appears at the top with one tab per
						configured locale.
					</li>
					<li>
						Click a non-default tab to author that locale's title, meta, and per-block content.
						Set the locale's status to "published" when ready.
					</li>
					<li>
						Build the site. The default locale renders at the base URL; each published locale
						overlay renders at <code class="font-mono text-[11.5px]">/&lt;locale&gt;/&lt;slug&gt;</code>
						with hreflang wired automatically.
					</li>
				</ol>
				<p class="mt-3 text-[12px] text-text-muted">
					Hreflang strategy (path vs subdomain) lives on the SEO settings page. Path is the safe
					default; subdomain mode requires DNS records for each locale subdomain.
				</p>
			</Card>
		</div>
	{/if}
</div>

<ConfirmDialog
	open={pendingDelete !== null}
	title="Remove this locale?"
	message={pendingDelete
		? `All page + block translations for "${pendingDelete}" will be removed. The default locale and the base page rows are unaffected. This cannot be undone.`
		: ''}
	confirmText="Remove"
	variant="danger"
	onconfirm={confirmDelete}
	oncancel={() => (pendingDelete = null)}
/>

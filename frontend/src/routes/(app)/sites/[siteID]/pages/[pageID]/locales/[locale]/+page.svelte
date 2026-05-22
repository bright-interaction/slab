<script lang="ts">
	import { goto } from '$app/navigation';
	import { page as pageState } from '$app/state';
	import * as pagesApi from '$lib/api/pages';
	import * as blocksApi from '$lib/api/blocks';
	import * as localesApi from '$lib/api/locales';
	import { ApiError } from '$lib/api/client';
	import Card from '$lib/components/ui/Card.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Textarea from '$lib/components/ui/Textarea.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { ArrowLeft } from 'lucide-svelte';
	import type { Block, Page } from '$lib/api/types';

	// Sprint 3 (multilingual v1, 2026-05-22): focused per-locale
	// editor for one page. Operator authors:
	//  - title + meta_title + meta_description + slug_override + status
	//    for this locale (page_locales overlay row)
	//  - one block_locales overlay per block, with the base data_json
	//    visible read-only as a reference and the locale's
	//    translated JSON in a textarea
	// Save dispatches an UpsertPageLocale + N UpsertBlockLocale calls;
	// errors surface inline and stop the save.

	const siteID = $derived(pageState.params.siteID as string);
	const pageID = $derived(pageState.params.pageID as string);
	const locale = $derived(pageState.params.locale as string);

	let basePage = $state<Page | null>(null);
	let blocks = $state<Block[]>([]);
	let pageLocale = $state<localesApi.PageLocale | null>(null);
	let blockLocales = $state<Record<string, localesApi.BlockLocale | undefined>>({});
	let blockDrafts = $state<Record<string, { dataJson: string; visible: boolean }>>({});
	let loading = $state(true);
	let saving = $state(false);

	let title = $state('');
	let metaTitle = $state('');
	let metaDescription = $state('');
	let slugOverride = $state('');
	let status = $state<'draft' | 'published' | 'archived'>('draft');

	const statusOptions = [
		{ value: 'draft', label: 'Draft' },
		{ value: 'published', label: 'Published' },
		{ value: 'archived', label: 'Archived' }
	];

	async function load() {
		loading = true;
		try {
			const [p, blocksRes, pls, bls, siteLocales] = await Promise.all([
				pagesApi.get(siteID, pageID),
				blocksApi.list(siteID, pageID),
				localesApi.listPageLocales(siteID, pageID),
				localesApi.listBlockLocalesByPage(siteID, pageID).catch(() => [] as localesApi.BlockLocale[]),
				localesApi.listSiteLocales(siteID).catch(() => [] as localesApi.SiteLocale[])
			]);
			basePage = p;
			blocks = blocksRes.blocks ?? [];
			pageLocale = pls.find((row) => row.locale === locale) ?? null;
			title = pageLocale?.title ?? '';
			metaTitle = pageLocale?.meta_title ?? '';
			metaDescription = pageLocale?.meta_description ?? '';
			slugOverride = pageLocale?.slug_override ?? '';
			status = (pageLocale?.status as typeof status) ?? 'draft';

			// Block locale rows for this locale only. Pull all from the
			// per-page batch call so a 60-block page doesn't fire 60
			// requests; then index by block_id for the draft state.
			const map: Record<string, localesApi.BlockLocale> = {};
			for (const row of bls) {
				if (row.locale === locale) {
					map[row.block_id] = row;
				}
			}
			blockLocales = map;
			const drafts: Record<string, { dataJson: string; visible: boolean }> = {};
			for (const bl of blocks) {
				const existing = map[bl.id];
				drafts[bl.id] = {
					dataJson: existing?.data_json && existing.data_json !== '{}' ? existing.data_json : '',
					visible: existing ? existing.is_visible : true
				};
			}
			blockDrafts = drafts;

			// Confirm locale is in the site's configured list. Otherwise
			// redirect to the locales settings page so the operator can
			// add it before trying to translate.
			if (siteLocales.length && !siteLocales.find((s) => s.locale === locale)) {
				toast.error(`Locale "${locale}" is not configured on this site. Add it in Settings first.`);
				await goto(`/sites/${siteID}/settings/locales`);
				return;
			}
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to load locale editor.');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	const blockOverrideCount = $derived(
		Object.values(blockDrafts).filter((d) => (d.dataJson ?? '').trim() !== '').length
	);

	async function save() {
		if (saving) return;
		saving = true;
		try {
			await localesApi.upsertPageLocale(siteID, pageID, locale, {
				title: title.trim(),
				meta_title: metaTitle.trim(),
				meta_description: metaDescription.trim(),
				slug_override: slugOverride.trim(),
				status
			});

			for (const bl of blocks) {
				const draft = blockDrafts[bl.id];
				const filled = (draft?.dataJson ?? '').trim();
				if (filled === '') {
					// Operator left this block untranslated; remove any
					// stale overlay row instead of writing an empty one.
					if (blockLocales[bl.id]) {
						await localesApi.deleteBlockLocale(siteID, bl.id, locale);
					}
					continue;
				}
				try {
					JSON.parse(filled);
				} catch {
					throw new Error(`Block "${bl.name || bl.block_type}" has invalid JSON in its translation. Fix the syntax and save again.`);
				}
				await localesApi.upsertBlockLocale(siteID, bl.id, locale, {
					data_json: filled,
					is_visible: draft.visible
				});
			}
			toast.success(`${locale} translation saved.`);
			await load();
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : err instanceof Error ? err.message : 'Save failed.';
			toast.error(msg);
		} finally {
			saving = false;
		}
	}

	function copyBaseInto(blockID: string, baseJson: string) {
		blockDrafts = {
			...blockDrafts,
			[blockID]: { ...blockDrafts[blockID], dataJson: baseJson }
		};
	}
</script>

<div class="mx-auto max-w-4xl px-6 py-8">
	<header class="flex items-start justify-between gap-4">
		<div class="flex flex-col gap-1.5">
			<a
				href={`/sites/${siteID}/pages/${pageID}`}
				class="inline-flex items-center gap-1.5 self-start text-[12px] text-text-muted hover:text-text-primary"
			>
				<ArrowLeft class="h-3.5 w-3.5" strokeWidth={1.5} />
				Back to page editor
			</a>
			<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
				Translate to <span class="font-mono">{locale}</span>
			</h1>
			{#if basePage}
				<p class="text-[13px] text-text-secondary">
					Source page: <span class="font-medium">{basePage.title}</span>
					<span class="font-mono text-[11.5px] text-text-muted">/{basePage.slug}</span>
				</p>
			{/if}
		</div>
		<div class="flex items-center gap-3">
			<Badge variant={status === 'published' ? 'success' : 'warning'}>{status}</Badge>
			<Button variant="primary" disabled={saving} onclick={save}>
				{saving ? 'Saving...' : 'Save'}
			</Button>
		</div>
	</header>

	{#if loading}
		<div class="mt-6 space-y-3">
			<Skeleton class="h-24" />
			<Skeleton class="h-32" />
			<Skeleton class="h-32" />
		</div>
	{:else if basePage}
		<div class="mt-6 flex flex-col gap-5">
			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Page metadata
				</h2>
				<p class="mt-1 text-[12px] text-text-muted">
					Empty fields fall back to the base page's values when the locale renders.
				</p>
				<div class="mt-4 grid grid-cols-1 gap-4 md:grid-cols-2">
					<Input label="Title" bind:value={title} placeholder={basePage.title} />
					<Input
						label="Slug override"
						bind:value={slugOverride}
						placeholder={`/${locale}/${basePage.slug}`}
					/>
					<Input label="Meta title" bind:value={metaTitle} placeholder={basePage.meta_title || basePage.title} />
					<Input
						label="Meta description"
						bind:value={metaDescription}
						placeholder={basePage.meta_description || ''}
					/>
				</div>
				<div class="mt-4 max-w-xs">
					<Select label="Status" options={statusOptions} bind:value={status} />
					<p class="mt-1.5 text-[11px] text-text-muted">
						The locale is only emitted to the built site when status is "published".
					</p>
				</div>
			</Card>

			<Card padding="md">
				<div class="flex items-center justify-between">
					<div>
						<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
							Block translations
						</h2>
						<p class="mt-1 text-[12px] text-text-muted">
							{blockOverrideCount} of {blocks.length} blocks translated. Blocks left empty render
							with the base content (visible to visitors as-is).
						</p>
					</div>
				</div>
				{#if blocks.length === 0}
					<p class="mt-4 text-[13px] text-text-muted">
						This page has no blocks yet. Add blocks on the main editor first.
					</p>
				{:else}
					<ul class="mt-4 flex flex-col gap-4">
						{#each blocks as bl (bl.id)}
							<li class="rounded-xl border border-border-light p-4">
								<div class="flex items-center justify-between gap-3">
									<div class="flex flex-col gap-0.5">
										<span class="text-[13px] font-medium text-text-primary">
											{bl.name || bl.block_type}
										</span>
										<span class="font-mono text-[11px] text-text-muted">{bl.block_type}</span>
									</div>
									{#if blockLocales[bl.id]}
										<Badge variant="success">Translated</Badge>
									{:else}
										<Badge variant="info">Not translated</Badge>
									{/if}
								</div>
								<div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2">
									<div>
										<label class="text-[11px] font-mono uppercase tracking-[0.18em] text-text-muted">
											Base content (read-only)
										</label>
										<pre class="mt-1 max-h-56 overflow-auto rounded-lg border border-border-light bg-bg-elevated p-3 font-mono text-[11.5px]">{bl.data_json}</pre>
									</div>
									<div>
										<div class="flex items-center justify-between">
											<label class="text-[11px] font-mono uppercase tracking-[0.18em] text-text-muted">
												{locale} translation
											</label>
											<button
												type="button"
												class="text-[11px] text-accent underline-offset-2 hover:underline"
												onclick={() => copyBaseInto(bl.id, bl.data_json)}
											>
												Copy from base
											</button>
										</div>
										<Textarea
											rows={10}
											placeholder="Leave empty to use the base content unchanged."
											value={blockDrafts[bl.id]?.dataJson ?? ''}
											oninput={(e) => {
												const value = (e.currentTarget as HTMLTextAreaElement).value;
												blockDrafts = {
													...blockDrafts,
													[bl.id]: { ...blockDrafts[bl.id], dataJson: value }
												};
											}}
										/>
										<p class="mt-1 text-[11px] text-text-muted">
											JSON with the same shape as the base. Field names stay the same; only the
											string values are translated.
										</p>
									</div>
								</div>
							</li>
						{/each}
					</ul>
				{/if}
			</Card>
		</div>
	{/if}
</div>

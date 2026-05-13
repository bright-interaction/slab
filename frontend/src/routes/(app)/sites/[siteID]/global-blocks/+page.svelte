<script lang="ts">
	import { page as pageState } from '$app/state';
	import * as globalBlocksApi from '$lib/api/globalBlocks';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import { Eye, EyeOff } from 'lucide-svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import { confirm } from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { Plus, Trash2, ChevronDown } from 'lucide-svelte';
	import type { GlobalBlock } from '$lib/api/types';

	const siteID = $derived(pageState.params.siteID as string);

	let blocks = $state<GlobalBlock[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	let creating = $state(false);
	let newName = $state('');
	let newSlot = $state<'header' | 'footer'>('header');
	let newType = $state('nav');
	let createOpen = $state(false);

	// Per-block draft state. Keyed by block id.
	let drafts = $state<Record<string, { name: string; data_json: string; dirty: boolean }>>({});
	let savingId = $state<Record<string, boolean>>({});
	let expandedId = $state<string | null>(null);

	// Per-block editor tab + lazy-loaded rendered HTML. The "Rendered" and
	// "Preview" tabs ask the server to render the CURRENT (possibly unsaved)
	// data_json so the operator sees live previews without committing.
	type ViewTab = 'data' | 'rendered' | 'preview';
	let viewTab = $state<Record<string, ViewTab>>({});
	let renderedHTML = $state<Record<string, string>>({});
	let renderingId = $state<Record<string, boolean>>({});
	let renderError = $state<Record<string, string | null>>({});

	const slotOptions = [
		{ value: 'header', label: 'Header' },
		{ value: 'footer', label: 'Footer' }
	];

	const blockTypeOptions = [
		{ value: 'nav', label: 'Nav (header)' },
		{ value: 'footer', label: 'Footer' },
		{ value: 'raw', label: 'Raw HTML' },
		{ value: 'banner', label: 'Banner' }
	];

	function prettyJSON(raw: string): string {
		try {
			const parsed = JSON.parse(raw || '{}');
			return JSON.stringify(parsed, null, 2);
		} catch {
			return raw;
		}
	}

	async function load() {
		loading = true;
		loadError = null;
		try {
			const res = await globalBlocksApi.list(siteID);
			blocks = res.global_blocks ?? [];
			drafts = Object.fromEntries(
				blocks.map((b) => [
					b.id,
					{
						name: b.name,
						// Pretty-print on load so editors aren't staring at one
						// giant minified line. Round-trip through saveOne re-
						// minifies (the backend stores compact); the Format
						// button below restores indentation on demand.
						data_json: prettyJSON(b.data_json || '{}'),
						dirty: false
					}
				])
			);
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load global blocks';
		} finally {
			loading = false;
		}
	}

	function formatDraft(id: string) {
		const current = drafts[id];
		if (!current) return;
		const formatted = prettyJSON(current.data_json);
		if (formatted === current.data_json) return;
		drafts = { ...drafts, [id]: { ...current, data_json: formatted, dirty: true } };
	}

	// Quick at-a-glance summary of what's in a global block , shown above
	// the JSON editor so the user knows where they are before reading the
	// raw payload. Per user feedback: "I want to be able to cast a fast
	// glance at it and know where I am editing."
	function blockSummary(b: GlobalBlock): string {
		try {
			const d = JSON.parse(b.data_json || '{}') as Record<string, unknown>;
			const parts: string[] = [];
			const brand = d.brand as Record<string, unknown> | undefined;
			if (brand) {
				const badge = typeof brand.badge === 'string' ? brand.badge : '';
				const wm = typeof brand.wordmark === 'string' ? brand.wordmark : '';
				const lbl = typeof brand.label === 'string' ? brand.label : '';
				const id = badge || wm || lbl;
				if (id) parts.push(`brand: ${id}`);
			}
			const links = d.links as unknown[] | undefined;
			if (Array.isArray(links)) {
				const ctaCount = links.filter(
					(l) => l && typeof l === 'object' && (l as Record<string, unknown>).cta === true
				).length;
				parts.push(
					ctaCount > 0
						? `${links.length} nav links (${ctaCount} CTA)`
						: `${links.length} nav links`
				);
			}
			const cols = d.columns as unknown[] | undefined;
			if (Array.isArray(cols)) parts.push(`${cols.length} footer columns`);
			const tagline = typeof d.tagline === 'string' ? d.tagline : '';
			if (tagline) parts.push(`tagline: ${tagline.slice(0, 40)}`);
			const copyright = typeof d.copyright === 'string' ? d.copyright : '';
			if (copyright) parts.push(`© ${copyright.slice(0, 40)}`);
			const theme = typeof d.theme === 'string' ? d.theme : '';
			if (theme) parts.push(`theme: ${theme}`);
			return parts.join(' · ') || 'empty data';
		} catch {
			return 'invalid JSON';
		}
	}

	$effect(() => {
		void load();
	});

	function setDraft(id: string, patch: Partial<{ name: string; data_json: string }>) {
		const current = drafts[id];
		if (!current) return;
		const next = { ...current, ...patch };
		next.dirty = true;
		drafts = { ...drafts, [id]: next };
	}

	async function saveOne(id: string) {
		const draft = drafts[id];
		if (!draft || !draft.dirty) return;
		// Validate JSON before sending.
		let parsedData: unknown = {};
		try {
			parsedData = draft.data_json.trim() === '' ? {} : JSON.parse(draft.data_json);
		} catch {
			toast.error('data JSON is not valid');
			return;
		}
		savingId = { ...savingId, [id]: true };
		try {
			const updated = await globalBlocksApi.update(siteID, id, {
				name: draft.name,
				data: parsedData
			});
			blocks = blocks.map((b) => (b.id === id ? updated : b));
			drafts = {
				...drafts,
				[id]: {
					name: updated.name,
					// Re-pretty so the editor stays readable across save cycles
					// (the backend stores compact JSON).
					data_json: prettyJSON(updated.data_json),
					dirty: false
				}
			};
			toast.success('Saved');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to save');
		} finally {
			const next = { ...savingId };
			delete next[id];
			savingId = next;
		}
	}

	async function toggleActive(b: GlobalBlock, active: boolean) {
		try {
			const updated = await globalBlocksApi.update(siteID, b.id, { is_active: active });
			blocks = blocks.map((x) => (x.id === b.id ? updated : x));
			toast.success(active ? `Enabled ${b.name}` : `Disabled ${b.name}`);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to toggle');
		}
	}

	async function removeOne(b: GlobalBlock) {
		const ok = await confirm({
			title: `Delete ${b.name}?`,
			message: 'Removing this global block will affect every page that uses this slot.',
			confirmText: 'Delete',
			cancelText: 'Keep',
			variant: 'danger'
		});
		if (!ok) return;
		try {
			await globalBlocksApi.remove(siteID, b.id);
			blocks = blocks.filter((x) => x.id !== b.id);
			const nextDrafts = { ...drafts };
			delete nextDrafts[b.id];
			drafts = nextDrafts;
			toast.success('Deleted');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to delete');
		}
	}

	// Fetches the rendered HTML for the block's CURRENT draft state. Falls
	// back to the saved data_json when no draft exists (the create path
	// doesn't seed drafts immediately).
	async function refreshRendered(block: GlobalBlock) {
		const draft = drafts[block.id];
		const data_json = draft?.data_json ?? block.data_json ?? '{}';
		// Validate JSON before round-tripping; render server will return
		// empty HTML for a broken payload, which we surface as an inline
		// error so the editor knows what's wrong.
		try {
			JSON.parse(data_json || '{}');
		} catch (err) {
			renderError = { ...renderError, [block.id]: 'data_json is not valid' };
			renderedHTML = { ...renderedHTML, [block.id]: '' };
			return;
		}
		renderingId = { ...renderingId, [block.id]: true };
		renderError = { ...renderError, [block.id]: null };
		try {
			const res = await globalBlocksApi.render(siteID, {
				block_type: block.block_type,
				slot: block.slot as 'header' | 'footer',
				data_json
			});
			renderedHTML = { ...renderedHTML, [block.id]: res.html ?? '' };
		} catch (err) {
			renderError = {
				...renderError,
				[block.id]: err instanceof Error ? err.message : 'Failed to render'
			};
			renderedHTML = { ...renderedHTML, [block.id]: '' };
		} finally {
			const next = { ...renderingId };
			delete next[block.id];
			renderingId = next;
		}
	}

	function switchTab(block: GlobalBlock, tab: ViewTab) {
		viewTab = { ...viewTab, [block.id]: tab };
		if (tab === 'rendered' || tab === 'preview') {
			void refreshRendered(block);
		}
	}

	// Pretty-print HTML with two-space indent + a newline after every
	// closing tag. Not a full HTML formatter (skips edge cases around
	// inline content + self-closing void elements with attributes); good
	// enough for the structured nav/footer output the renderer emits.
	function prettyHTML(html: string): string {
		if (!html) return '';
		// Insert linebreaks between adjacent tags so the indenter has
		// something to work with.
		let s = html.replace(/></g, '>\n<');
		const lines = s.split('\n');
		let depth = 0;
		const VOID = new Set([
			'area', 'base', 'br', 'col', 'embed', 'hr', 'img', 'input',
			'link', 'meta', 'source', 'track', 'wbr'
		]);
		return lines
			.map((line) => {
				const t = line.trim();
				if (!t) return '';
				const isClose = /^<\//.test(t);
				const tagMatch = t.match(/^<\/?([a-zA-Z][a-zA-Z0-9-]*)/);
				const tag = tagMatch && tagMatch[1] ? tagMatch[1].toLowerCase() : '';
				const isVoid = VOID.has(tag) || /\/>$/.test(t);
				const isSelfContained = /^<([a-zA-Z][a-zA-Z0-9-]*)[^>]*>.*<\/\1>$/.test(t);
				if (isClose) depth = Math.max(0, depth - 1);
				const out = '  '.repeat(depth) + t;
				if (!isClose && !isVoid && !isSelfContained) depth += 1;
				return out;
			})
			.filter(Boolean)
			.join('\n');
	}

	async function createOne() {
		if (!newName.trim()) {
			toast.error('Name is required');
			return;
		}
		creating = true;
		try {
			const created = await globalBlocksApi.create(siteID, {
				name: newName.trim(),
				slot: newSlot,
				block_type: newType,
				data: {}
			});
			blocks = [...blocks, created];
			drafts = {
				...drafts,
				[created.id]: { name: created.name, data_json: created.data_json, dirty: false }
			};
			expandedId = created.id;
			newName = '';
			createOpen = false;
			toast.success(`Created ${created.name}`);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to create');
		} finally {
			creating = false;
		}
	}

	const headerBlocks = $derived(blocks.filter((b) => b.slot === 'header'));
	const footerBlocks = $derived(blocks.filter((b) => b.slot === 'footer'));
	const slotGroups = $derived([
		{ slot: 'header', label: 'Header', list: headerBlocks },
		{ slot: 'footer', label: 'Footer', list: footerBlocks }
	]);
	const dataJsonHint =
		'Example: {"links": [{"label": "Pricing", "href": "/pricing"}]}';
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex flex-wrap items-start justify-between gap-4">
		<div>
			<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
				Global blocks
			</h1>
			<p class="mt-1 text-[13px] text-text-secondary">
				Site-wide header and footer slots. Edit once, applies to every page.
				Per-page suppression is available in the page editor's SEO panel for
				landing-page conversions that shouldn't carry the nav.
			</p>
		</div>
		<Button variant="primary" onclick={() => (createOpen = !createOpen)}>
			<Plus class="mr-1 h-3.5 w-3.5" />
			New global block
		</Button>
	</header>

	{#if createOpen}
		<Card padding="md" class="mt-6">
			<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
				<Input label="Name" bind:value={newName} placeholder="Site header" />
				<div class="flex flex-col gap-1.5">
					<span class="text-[12px] font-medium text-text-secondary">Slot</span>
					<Select
						options={slotOptions}
						value={newSlot}
						onchange={(v) => (newSlot = v as 'header' | 'footer')}
					/>
				</div>
				<div class="flex flex-col gap-1.5">
					<span class="text-[12px] font-medium text-text-secondary">Block type</span>
					<Select options={blockTypeOptions} bind:value={newType} />
				</div>
			</div>
			<div class="mt-4 flex justify-end gap-2 border-t border-border-light pt-4">
				<Button variant="ghost" onclick={() => (createOpen = false)}>Cancel</Button>
				<Button variant="primary" loading={creating} onclick={createOne}>Create</Button>
			</div>
		</Card>
	{/if}

	{#if loading}
		<div class="mt-8 space-y-3">
			<Skeleton width="100%" height="3rem" />
			<Skeleton width="100%" height="3rem" />
		</div>
	{:else if loadError}
		<Card padding="md" class="mt-8">
			<p class="text-[13px] text-danger">{loadError}</p>
		</Card>
	{:else if blocks.length === 0}
		<Card padding="none" class="mt-8">
			<EmptyState
				title="No global blocks yet"
				description="Add a header or footer block to render across every page on this site."
			>
				{#snippet action()}
					<Button variant="primary" onclick={() => (createOpen = true)}>
						<Plus class="mr-1 h-3.5 w-3.5" />
						New global block
					</Button>
				{/snippet}
			</EmptyState>
		</Card>
	{:else}
		<div class="mt-8 flex flex-col gap-8">
			{#each slotGroups as group (group.slot)}
				<section class="flex flex-col gap-3">
					<div class="flex items-baseline justify-between">
						<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
							{group.label}
						</h2>
						<span class="text-[11px] text-text-muted">
							{group.list.length}
							{group.list.length === 1 ? 'block' : 'blocks'}
						</span>
					</div>
					{#if group.list.length === 0}
						<p class="rounded-xl border border-dashed border-border-light bg-bg-surface px-5 py-4 text-center text-[12.5px] text-text-muted">
							No {group.label.toLowerCase()} block. Click "New global block" to add one.
						</p>
					{/if}
					{#each group.list as block (block.id)}
						{@const draft = drafts[block.id]}
						<div class="rounded-xl border border-border-light bg-bg-surface">
							<div class="flex items-center gap-3 px-4 py-3">
								<button
									type="button"
									class="flex flex-1 items-center gap-3 overflow-hidden text-left"
									onclick={() => (expandedId = expandedId === block.id ? null : block.id)}
								>
									<span
										class="inline-flex h-5 shrink-0 items-center rounded-md bg-bg-elevated px-1.5 font-mono text-[10px] uppercase tracking-wider text-text-secondary"
									>
										{block.block_type}
									</span>
									<span class="shrink-0 text-[13px] text-text-primary">{block.name}</span>
									<span class="truncate text-[12px] text-text-muted">
										{blockSummary(block)}
									</span>
									{#if draft?.dirty}
										<span class="shrink-0 text-[10.5px] text-accent">unsaved</span>
									{/if}
								</button>
								<div class="flex items-center gap-2">
									<button
										type="button"
										title={block.is_active === 1 ? 'Disable on built site' : 'Enable on built site'}
										aria-label={block.is_active === 1
											? 'Disable global block'
											: 'Enable global block'}
										class="inline-flex h-7 items-center gap-1 rounded-md px-2 text-[12px] transition-colors {block.is_active ===
										1
											? 'text-text-primary hover:bg-bg-hover'
											: 'text-text-muted hover:bg-bg-hover hover:text-text-primary'}"
										onclick={() => toggleActive(block, block.is_active !== 1)}
									>
										{#if block.is_active === 1}
											<Eye class="h-3.5 w-3.5" />
											Active
										{:else}
											<EyeOff class="h-3.5 w-3.5" />
											Inactive
										{/if}
									</button>
									<button
										type="button"
										aria-label="Delete block"
										class="inline-flex h-7 w-7 items-center justify-center rounded-md text-text-muted hover:bg-danger/10 hover:text-danger"
										onclick={() => removeOne(block)}
									>
										<Trash2 class="h-3.5 w-3.5" />
									</button>
									<button
										type="button"
										class="inline-flex h-7 w-7 items-center justify-center rounded-md text-text-muted hover:bg-bg-hover hover:text-text-primary"
										aria-label={expandedId === block.id ? 'Collapse' : 'Expand'}
										onclick={() => (expandedId = expandedId === block.id ? null : block.id)}
									>
										<ChevronDown
											class="h-4 w-4 transition-transform {expandedId === block.id
												? 'rotate-180'
												: ''}"
										/>
									</button>
								</div>
							</div>
							{#if expandedId === block.id && draft}
								{@const tab = viewTab[block.id] ?? 'data'}
								<div class="flex flex-col gap-4 border-t border-border-light px-4 py-4">
									<Input
										label="Name"
										value={draft.name}
										oninput={(e) =>
											setDraft(block.id, {
												name: (e.currentTarget as HTMLInputElement).value
											})}
									/>

									<!-- Tabbed editor: structured data (JSON) on the left,
									     rendered HTML output in the middle, sandboxed preview
									     on the right. Data is the editable surface; Rendered
									     and Preview are read-only views of what the builder
									     would emit at deploy time. -->
									<div class="flex items-center gap-1 border-b border-border-light">
										{#each [{ id: 'data', label: 'Data (JSON)' }, { id: 'rendered', label: 'Rendered HTML' }, { id: 'preview', label: 'Preview' }] as t (t.id)}
											<button
												type="button"
												onclick={() => switchTab(block, t.id as ViewTab)}
												class="relative px-3 py-1.5 text-[12px] transition-colors {tab === t.id
													? 'text-text-primary'
													: 'text-text-muted hover:text-text-primary'}"
											>
												{t.label}
												{#if tab === t.id}
													<span class="absolute inset-x-0 -bottom-px h-0.5 bg-text-primary"></span>
												{/if}
											</button>
										{/each}
									</div>

									{#if tab === 'data'}
										<div class="flex flex-col gap-1.5">
											<div class="flex items-center justify-between">
												<label
													for="data-json-{block.id}"
													class="text-[12px] font-medium text-text-secondary"
												>
													Block data (JSON)
												</label>
												<button
													type="button"
													class="text-[11px] text-text-muted hover:text-text-primary"
													onclick={() => formatDraft(block.id)}
													title="Re-indent JSON"
												>
													Format
												</button>
											</div>
											<textarea
												id="data-json-{block.id}"
												rows={20}
												class="w-full rounded-md border border-border bg-bg-elevated px-3 py-2 font-mono text-[12.5px] leading-[1.55] text-text-primary outline-none transition-colors focus:border-text-primary"
												value={draft.data_json}
												oninput={(e) =>
													setDraft(block.id, {
														data_json: (e.currentTarget as HTMLTextAreaElement).value
													})}
												spellcheck="false"
											></textarea>
											<p class="text-[11px] text-text-muted">
												{dataJsonHint}. Tip: header blocks accept brand
												({`{badge, wordmark, accent_from, href}`}) and links
												({`[{label, href, cta?}]`}); footer blocks accept columns,
												tagline, copyright, theme.
											</p>
										</div>
									{:else if tab === 'rendered'}
										<div class="flex flex-col gap-1.5">
											<div class="flex items-center justify-between">
												<span class="text-[12px] font-medium text-text-secondary">
													Rendered HTML (read-only)
												</span>
												<button
													type="button"
													class="text-[11px] text-text-muted hover:text-text-primary"
													onclick={() => refreshRendered(block)}
													disabled={Boolean(renderingId[block.id])}
													title="Re-render with current draft data"
												>
													{renderingId[block.id] ? 'Rendering…' : 'Refresh'}
												</button>
											</div>
											{#if renderError[block.id]}
												<p class="text-[12px] text-danger">{renderError[block.id]}</p>
											{:else if renderingId[block.id] && !renderedHTML[block.id]}
												<p class="text-[12px] text-text-muted">Rendering…</p>
											{:else if !renderedHTML[block.id]}
												<p class="text-[12px] text-text-muted">
													(empty - the renderer returned no output for this data)
												</p>
											{:else}
												{@const html = prettyHTML(renderedHTML[block.id] ?? '')}
												{@const lines = html.split('\n')}
												<div class="overflow-auto rounded-md border border-border bg-bg-elevated">
													<pre class="m-0 px-3 py-2 font-mono text-[12.5px] leading-[1.55]"><code>{#each lines as line, i (i)}<span class="block"><span class="mr-3 inline-block w-8 select-none text-right text-text-muted">{i + 1}</span><span class="text-text-primary">{line || ' '}</span></span>{/each}</code></pre>
												</div>
												<p class="text-[11px] text-text-muted">
													This is exactly what the builder writes into the deployed Astro layout.
													Edit the Data tab to change it; this view re-renders on Refresh and on tab switch.
												</p>
											{/if}
										</div>
									{:else if tab === 'preview'}
										<div class="flex flex-col gap-1.5">
											<div class="flex items-center justify-between">
												<span class="text-[12px] font-medium text-text-secondary">
													Visual preview (sandboxed iframe)
												</span>
												<button
													type="button"
													class="text-[11px] text-text-muted hover:text-text-primary"
													onclick={() => refreshRendered(block)}
													disabled={Boolean(renderingId[block.id])}
												>
													{renderingId[block.id] ? 'Rendering…' : 'Refresh'}
												</button>
											</div>
											{#if renderError[block.id]}
												<p class="text-[12px] text-danger">{renderError[block.id]}</p>
											{:else if renderingId[block.id] && !renderedHTML[block.id]}
												<p class="text-[12px] text-text-muted">Rendering…</p>
											{:else if !renderedHTML[block.id]}
												<p class="text-[12px] text-text-muted">
													(empty - no HTML to preview)
												</p>
											{:else}
												<iframe
													title="Block preview"
													sandbox=""
													class="h-[260px] w-full rounded-md border border-border bg-white"
													srcdoc={`<!doctype html><html><head><meta charset="utf-8"><style>body{margin:0;font-family:Inter,system-ui,sans-serif;color:#0a0a0a;background:#fafafa;padding:16px}.site-header,.site-footer{background:#fff;border:1px solid #e5e5e5;border-radius:8px;padding:12px 16px}.site-header .container,.site-footer .container{display:flex;align-items:center;justify-content:space-between;gap:16px;flex-wrap:wrap}a{color:#0a0a0a;text-decoration:none;font-size:14px}a:hover{text-decoration:underline}.brand-mark{display:inline-flex;align-items:center;gap:8px;font-weight:600}.brand-badge{background:#171717;color:#fff;width:24px;height:24px;border-radius:6px;display:inline-flex;align-items:center;justify-content:center;font-size:11px}</style></head><body>${renderedHTML[block.id]}</body></html>`}
												></iframe>
												<p class="text-[11px] text-text-muted">
													Approximates the deployed site's styling. Final appearance depends on the
													site's CSS theme; this preview uses a neutral baseline.
												</p>
											{/if}
										</div>
									{/if}

									<div class="flex justify-end gap-2 border-t border-border-light pt-3">
										<Button
											variant="ghost"
											onclick={() => {
												drafts = {
													...drafts,
													[block.id]: {
														name: block.name,
														data_json: prettyJSON(block.data_json || '{}'),
														dirty: false
													}
												};
											}}
											disabled={!draft.dirty}
										>
											Reset
										</Button>
										<Button
											variant="primary"
											loading={Boolean(savingId[block.id])}
											disabled={!draft.dirty}
											onclick={() => saveOne(block.id)}
										>
											Save
										</Button>
									</div>
								</div>
							{/if}
						</div>
					{/each}
				</section>
			{/each}
		</div>
	{/if}
</div>

<script lang="ts">
	import { page as pageState } from '$app/state';
	import * as pagesApi from '$lib/api/pages';
	import * as blocksApi from '$lib/api/blocks';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Textarea from '$lib/components/ui/Textarea.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Switch from '$lib/components/ui/Switch.svelte';
	import { confirm } from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import BlockEditor from '$lib/components/pages/BlockEditor.svelte';
	import PageSourceDialog from '$lib/components/pages/PageSourceDialog.svelte';
	import TextMode from '$lib/components/pages/TextMode.svelte';
	import { ChevronDown, ArrowLeft, Code, Type, LayoutPanelTop } from 'lucide-svelte';
	import type { Block, Page } from '$lib/api/types';

	const siteID = $derived(pageState.params.siteID as string);
	const pageID = $derived(pageState.params.pageID as string);

	let pageData = $state<Page | null>(null);
	let blocks = $state<Block[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	let editorState = $state<{ blocks: Block[]; dirty: boolean }>({ blocks: [], dirty: false });

	// Inline title editor
	let titleEditing = $state(false);
	let titleDraft = $state('');
	let titleSaving = $state(false);

	// SEO panel state
	let seoOpen = $state(false);
	let seoMetaTitle = $state('');
	let seoMetaDescription = $state('');
	let seoOgImage = $state('');
	let seoCanonical = $state('');
	let seoNoIndex = $state(false);
	let seoLayout = $state('default');
	let seoSaving = $state(false);
	let seoDirty = $state(false);
	let seoBaseline = $state({
		meta_title: '',
		meta_description: '',
		og_image_id: '',
		canonical_url: '',
		no_index: false,
		layout: 'default'
	});

	let saving = $state(false);
	let editorRef = $state<{
		reset: () => void;
		getBlocks: () => Block[];
		syncFrom: (next: Block[]) => void;
	} | null>(null);

	// Editing surface. "Blocks" is the canonical full editor, "Text" is the
	// content-writer view (inline-editable text fields per block, autosave on
	// blur). The </> source dialog is opened on demand and works in both.
	let editorMode = $state<'blocks' | 'text'>('blocks');
	let sourceOpen = $state(false);
	// Per-block inline-text save state for the Text Mode UI. Keyed by block id.
	let textSaving = $state<Record<string, boolean>>({});

	const layoutOptions = [
		{ value: 'default', label: 'Default' },
		{ value: 'landing', label: 'Landing' },
		{ value: 'legal', label: 'Legal' },
		{ value: 'minimal', label: 'Minimal' }
	];

	async function loadAll() {
		loading = true;
		loadError = null;
		try {
			const [p, blocksRes] = await Promise.all([
				pagesApi.get(siteID, pageID),
				blocksApi.list(siteID, pageID)
			]);
			pageData = p;
			blocks = [...(blocksRes.blocks ?? [])].sort((a, b) => a.sort_order - b.sort_order);
			titleDraft = p.title;
			hydrateSeo(p);
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load page';
		} finally {
			loading = false;
		}
	}

	function hydrateSeo(p: Page) {
		seoMetaTitle = p.meta_title || '';
		seoMetaDescription = p.meta_description || '';
		seoOgImage = p.og_image_id || '';
		seoCanonical = p.canonical_url || '';
		seoNoIndex = p.no_index === 1;
		seoLayout = p.layout || 'default';
		seoBaseline = {
			meta_title: seoMetaTitle,
			meta_description: seoMetaDescription,
			og_image_id: seoOgImage,
			canonical_url: seoCanonical,
			no_index: seoNoIndex,
			layout: seoLayout
		};
		seoDirty = false;
	}

	$effect(() => {
		void loadAll();
	});

	$effect(() => {
		seoDirty =
			seoMetaTitle !== seoBaseline.meta_title ||
			seoMetaDescription !== seoBaseline.meta_description ||
			seoOgImage !== seoBaseline.og_image_id ||
			seoCanonical !== seoBaseline.canonical_url ||
			seoNoIndex !== seoBaseline.no_index ||
			seoLayout !== seoBaseline.layout;
	});

	function startTitleEdit() {
		if (!pageData) return;
		titleDraft = pageData.title;
		titleEditing = true;
	}

	async function commitTitle() {
		if (!pageData) return;
		const next = titleDraft.trim();
		if (next === pageData.title || next.length < 2) {
			titleEditing = false;
			titleDraft = pageData.title;
			return;
		}
		titleSaving = true;
		try {
			const updated = await pagesApi.update(siteID, pageID, { title: next });
			pageData = updated;
			titleEditing = false;
			toast.success('Title updated.');
		} catch (err) {
			const msg = err instanceof Error ? err.message : 'Failed to update title';
			toast.error(msg);
		} finally {
			titleSaving = false;
		}
	}

	function handleTitleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			void commitTitle();
		} else if (e.key === 'Escape') {
			titleEditing = false;
			if (pageData) titleDraft = pageData.title;
		}
	}

	async function saveSeo() {
		if (!seoDirty) return;
		seoSaving = true;
		try {
			const updated = await pagesApi.update(siteID, pageID, {
				meta_title: seoMetaTitle,
				meta_description: seoMetaDescription,
				og_image_id: seoOgImage,
				canonical_url: seoCanonical,
				no_index: seoNoIndex,
				layout: seoLayout
			});
			pageData = updated;
			hydrateSeo(updated);
			toast.success('SEO settings saved.');
		} catch (err) {
			const msg = err instanceof Error ? err.message : 'Failed to save SEO';
			toast.error(msg);
		} finally {
			seoSaving = false;
		}
	}

	async function saveBlocks() {
		if (!editorState.dirty) return;
		saving = true;
		try {
			const payload = editorState.blocks.map((b, i) => ({
				id: b.id || undefined,
				block_type: b.block_type,
				sort_order: i,
				data: safeParseJson(b.data_json),
				style: safeParseJson(b.style_json),
				is_visible: b.is_visible === 1
			}));
			const res = await blocksApi.bulkSave(siteID, pageID, payload);
			const next = [...(res.blocks ?? [])].sort((a, b) => a.sort_order - b.sort_order);
			blocks = next;
			editorRef?.syncFrom(next);
			toast.success('Page saved.');
		} catch (err) {
			const msg = err instanceof Error ? err.message : 'Failed to save page';
			toast.error(msg);
		} finally {
			saving = false;
		}
	}

	function safeParseJson(input: string): unknown {
		if (!input || input.trim() === '') return {};
		try {
			return JSON.parse(input);
		} catch {
			return {};
		}
	}

	// Inline-text save from the Text Mode surface. PATCHes a single field on
	// a block's data_json and reflects the result back into local state so
	// the BlockEditor sees the same value if the user toggles back. Targets
	// the existing PATCH /api/sites/{id}/pages/{id}/blocks/{id} endpoint,
	// no new mutation API is needed.
	async function handleTextField(blockID: string, fieldKey: string, nextValue: string) {
		const current = blocks.find((b) => b.id === blockID);
		if (!current) return;
		const data = (() => {
			try {
				const v = JSON.parse(current.data_json || '{}');
				return v && typeof v === 'object' && !Array.isArray(v) ? (v as Record<string, unknown>) : {};
			} catch {
				return {} as Record<string, unknown>;
			}
		})();
		const previous = typeof data[fieldKey] === 'string' ? (data[fieldKey] as string) : '';
		if (nextValue === previous) return;
		const nextData = { ...data, [fieldKey]: nextValue };
		textSaving = { ...textSaving, [blockID]: true };
		try {
			const updated = await blocksApi.update(siteID, pageID, blockID, { data: nextData });
			blocks = blocks.map((b) => (b.id === blockID ? updated : b));
			editorRef?.syncFrom(blocks);
			toast.success(`Saved ${fieldKey}`);
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to save text');
		} finally {
			const next = { ...textSaving };
			delete next[blockID];
			textSaving = next;
		}
	}

	async function discardBlocks() {
		if (!editorState.dirty) return;
		const ok = await confirm({
			title: 'Discard unsaved block changes?',
			message: 'You will lose any edits made since the last save.',
			confirmText: 'Discard',
			cancelText: 'Keep editing',
			variant: 'danger'
		});
		if (!ok) return;
		editorRef?.reset();
		toast.info('Changes discarded.');
	}

	function metaTitleCount(): number {
		return seoMetaTitle.length;
	}

	function metaDescCount(): number {
		return seoMetaDescription.length;
	}

	function focusOnMount(node: HTMLInputElement) {
		queueMicrotask(() => node.focus());
	}

	function formatDate(ts: string): string {
		if (!ts) return '';
		const d = new Date(ts);
		if (Number.isNaN(d.getTime())) return '';
		return d.toLocaleString(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<div class="mb-4">
		<a
			href={`/sites/${siteID}/pages`}
			class="inline-flex items-center gap-1.5 text-[12px] text-text-muted transition-colors hover:text-text-primary"
		>
			<ArrowLeft class="h-3.5 w-3.5" />
			All pages
		</a>
	</div>

	{#if loading}
		<div class="space-y-4">
			<Skeleton width="40%" height="2rem" />
			<Skeleton width="20%" height="0.9rem" />
			<Card padding="md">
				<Skeleton width="100%" height="6rem" />
			</Card>
		</div>
	{:else if loadError}
		<Card padding="md">
			<p class="text-[13px] text-danger">{loadError}</p>
		</Card>
	{:else if pageData}
		<header class="flex items-start justify-between gap-4">
			<div class="min-w-0 flex-1">
				{#if titleEditing}
					<input
						class="w-full bg-transparent font-display text-3xl font-extralight tracking-tight text-text-primary placeholder:text-text-muted focus:outline-none"
						bind:value={titleDraft}
						onblur={commitTitle}
						onkeydown={handleTitleKeydown}
						disabled={titleSaving}
						use:focusOnMount
					/>
				{:else}
					<button
						type="button"
						class="block w-full text-left"
						onclick={startTitleEdit}
						title="Click to edit title"
					>
						<h1
							class="font-display text-3xl font-extralight tracking-tight text-text-primary hover:text-accent"
						>
							{pageData.title || 'Untitled'}
						</h1>
					</button>
				{/if}
				<div class="mt-1 flex items-center gap-3 text-[12px] text-text-muted">
					<span class="font-mono">/{pageData.slug}</span>
					<span>·</span>
					<span>updated {formatDate(pageData.updated_at)}</span>
					<span>·</span>
					<span class="capitalize">{pageData.status || 'draft'}</span>
				</div>
			</div>
			<div class="flex shrink-0 items-center gap-2">
				<Button
					variant="ghost"
					onclick={() => (sourceOpen = true)}
					title="View the rendered .astro source for this page"
				>
					<Code class="mr-1 h-3.5 w-3.5" />
					View source
				</Button>
				<Button
					variant="ghost"
					disabled={!editorState.dirty || saving}
					onclick={discardBlocks}
				>
					Discard
				</Button>
				<Button
					variant="primary"
					loading={saving}
					disabled={!editorState.dirty}
					onclick={saveBlocks}
				>
					{editorState.dirty ? 'Save changes' : 'Saved'}
				</Button>
			</div>
		</header>

		<section class="mt-6">
			<button
				type="button"
				class="flex w-full items-center justify-between rounded-xl border border-border-light bg-bg-surface px-4 py-3 text-left transition-colors hover:bg-bg-elevated/40"
				onclick={() => (seoOpen = !seoOpen)}
			>
				<div>
					<p class="text-[13px] font-medium text-text-primary">SEO and metadata</p>
					<p class="text-[11px] text-text-muted">
						Meta title, description, indexing, layout. {seoDirty ? 'Unsaved.' : ''}
					</p>
				</div>
				<ChevronDown
					class="h-4 w-4 text-text-muted transition-transform {seoOpen ? 'rotate-180' : ''}"
				/>
			</button>

			{#if seoOpen}
				<div class="mt-2 rounded-xl border border-border-light bg-bg-surface p-5">
					<div class="grid grid-cols-1 gap-4 md:grid-cols-2">
						<Input
							label="Meta title"
							value={seoMetaTitle}
							oninput={(e) => {
								seoMetaTitle = (e.currentTarget as HTMLInputElement).value;
							}}
							hint={`${metaTitleCount()} characters. 50-60 recommended.`}
						/>
						<Input
							label="Canonical URL"
							value={seoCanonical}
							oninput={(e) => {
								seoCanonical = (e.currentTarget as HTMLInputElement).value;
							}}
							placeholder="https://example.com/page"
						/>
						<div class="md:col-span-2">
							<Textarea
								label="Meta description"
								rows={3}
								value={seoMetaDescription}
								oninput={(e) => {
									seoMetaDescription = (e.currentTarget as HTMLTextAreaElement).value;
								}}
								hint={`${metaDescCount()} characters. 140-160 recommended.`}
							/>
						</div>
						<Input
							label="OG image ID"
							value={seoOgImage}
							oninput={(e) => {
								seoOgImage = (e.currentTarget as HTMLInputElement).value;
							}}
							hint="Media picker arrives in commit 11."
						/>
						<div class="flex flex-col gap-1.5">
							<span class="text-[12px] font-medium text-text-secondary">Layout</span>
							<Select options={layoutOptions} bind:value={seoLayout} />
						</div>
						<div class="flex items-center justify-between md:col-span-2">
							<div>
								<p class="text-[13px] font-medium text-text-primary">No index</p>
								<p class="text-[11px] text-text-muted">
									Hide this page from search engines and the sitemap.
								</p>
							</div>
							<Switch bind:checked={seoNoIndex} />
						</div>
					</div>
					<div class="mt-4 flex justify-end gap-2 border-t border-border-light pt-4">
						<Button
							variant="ghost"
							disabled={!seoDirty || seoSaving}
							onclick={() => pageData && hydrateSeo(pageData)}
						>
							Reset
						</Button>
						<Button
							variant="primary"
							loading={seoSaving}
							disabled={!seoDirty}
							onclick={saveSeo}
						>
							Save SEO
						</Button>
					</div>
				</div>
			{/if}
		</section>

		<section class="mt-8">
			<div class="mb-3 flex items-baseline justify-between gap-3">
				<div
					role="group"
					aria-label="Editor mode"
					class="inline-flex items-center gap-0.5 rounded-md border border-border-light bg-bg-elevated p-0.5"
				>
					<button
						type="button"
						aria-pressed={editorMode === 'blocks'}
						onclick={() => (editorMode = 'blocks')}
						class="inline-flex h-7 items-center gap-1.5 rounded px-2.5 text-[11.5px] font-medium transition-colors {editorMode ===
						'blocks'
							? 'bg-bg-surface text-text-primary shadow-sm'
							: 'text-text-muted hover:text-text-primary'}"
					>
						<LayoutPanelTop class="h-3.5 w-3.5" />
						Blocks
					</button>
					<button
						type="button"
						aria-pressed={editorMode === 'text'}
						onclick={() => (editorMode = 'text')}
						title="Edit text fields inline. Saves on blur."
						class="inline-flex h-7 items-center gap-1.5 rounded px-2.5 text-[11.5px] font-medium transition-colors {editorMode ===
						'text'
							? 'bg-bg-surface text-text-primary shadow-sm'
							: 'text-text-muted hover:text-text-primary'}"
					>
						<Type class="h-3.5 w-3.5" />
						Text
					</button>
				</div>
				<span class="text-[11px] text-text-muted">
					{blocks.length} block(s){editorMode === 'blocks' && editorState.dirty ? ' · unsaved' : ''}
					{#if editorMode === 'text' && Object.keys(textSaving).length > 0}
						· saving…
					{/if}
				</span>
			</div>
			{#if editorMode === 'blocks'}
				<BlockEditor
					bind:this={editorRef}
					{siteID}
					{pageID}
					initialBlocks={blocks}
					onChange={(state) => {
						editorState = state;
					}}
				/>
			{:else}
				<TextMode {blocks} onFieldChange={handleTextField} />
			{/if}
		</section>
	{/if}
</div>

<PageSourceDialog bind:open={sourceOpen} {siteID} {pageID} />


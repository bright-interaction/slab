<script lang="ts">
	import { untrack } from 'svelte';
	import { GripVertical, ChevronDown, Trash2, Code, Copy, Check } from 'lucide-svelte';
	import Switch from '$lib/components/ui/Switch.svelte';
	import BlockTypeForm from './BlockTypeForm.svelte';
	import * as blocksApi from '$lib/api/blocks';
	import { toast } from '$lib/stores/toast.svelte';
	import type { Block } from '$lib/api/types';

	let {
		block,
		siteID,
		pageID,
		expanded = false,
		dragging = false,
		onToggleExpanded,
		onToggleVisibility,
		onDataChange,
		onDelete,
		ondragstart,
		ondragover,
		ondragend,
		ondrop
	}: {
		block: Block;
		siteID: string;
		pageID: string;
		expanded?: boolean;
		dragging?: boolean;
		onToggleExpanded: () => void;
		onToggleVisibility: (visible: boolean) => void;
		onDataChange: (nextDataJson: string) => void;
		onDelete: () => void;
		ondragstart?: (e: DragEvent) => void;
		ondragover?: (e: DragEvent) => void;
		ondragend?: (e: DragEvent) => void;
		ondrop?: (e: DragEvent) => void;
	} = $props();

	// Code preview state. Lazy: only fetched the first time the </> toggle
	// opens or after a save (which dirties the cached source).
	let codeOpen = $state(false);
	let codeLoading = $state(false);
	let codeError = $state<string | null>(null);
	let codeAstro = $state<string | null>(null);
	let codeBlockType = $state<string>('');
	let copied = $state(false);

	async function loadCode(): Promise<void> {
		if (!block.id) {
			codeError = 'Save the page once before viewing source. New blocks need an ID.';
			return;
		}
		codeLoading = true;
		codeError = null;
		try {
			const res = await blocksApi.preview(siteID, pageID, block.id);
			codeAstro = res.astro;
			codeBlockType = res.block_type;
		} catch (err) {
			codeError = err instanceof Error ? err.message : 'Failed to render block';
		} finally {
			codeLoading = false;
		}
	}

	function toggleCode(): void {
		codeOpen = !codeOpen;
		if (codeOpen && codeAstro === null) {
			void loadCode();
		}
	}

	async function refreshCode(): Promise<void> {
		codeAstro = null;
		await loadCode();
	}

	async function copyCode(): Promise<void> {
		if (!codeAstro) return;
		try {
			await navigator.clipboard.writeText(codeAstro);
			copied = true;
			toast.success('Source copied');
			setTimeout(() => (copied = false), 1500);
		} catch {
			toast.error('Could not copy');
		}
	}

	// Re-fetch when the underlying data changes after the panel was opened
	// at least once. Otherwise the code would lag behind a fresh edit.
	// CRITICAL: Svelte 5 $effect tracks every reactive read in its body.
	// Reading codeAstro/codeOpen/codeLoading inside the effect would make
	// it self-trigger when those state vars change inside refreshCode
	// (codeAstro flips null → string → re-runs effect → re-fetches → loop).
	// Only block.data_json should be reactive here; the rest read via
	// untrack so the effect only fires on actual data changes.
	$effect(() => {
		void block.data_json;
		untrack(() => {
			if (codeOpen && codeAstro !== null && !codeLoading) {
				void refreshCode();
			}
		});
	});

	let visibleBound = $state(false);
	let visibleInit = $state(false);

	$effect(() => {
		const next = block.is_visible === 1;
		if (!visibleInit) {
			visibleInit = true;
			visibleBound = next;
			return;
		}
		if (next !== visibleBound) {
			visibleBound = next;
		}
	});

	$effect(() => {
		const expected = block.is_visible === 1;
		if (visibleInit && visibleBound !== expected) {
			onToggleVisibility(visibleBound);
		}
	});

	function summary(): string {
		try {
			const parsed = JSON.parse(block.data_json || '{}') as Record<string, unknown>;
			// Try the most-distinctive text field for each block type. Order
			// matters: prefer the user-visible headline over generic labels
			// when both exist (hero blocks have 'headline' AND a 'subheading').
			const candidates = [
				'headline', // hero, split_hero
				'heading', // stat_grid, replacement_grid, process_steps, pricing, about_split, accordion_faq, cta, feature_grid, code_block, form
				'title', // legacy + custom
				'label', // logo_strip, logo_carousel
				'name', // pricing tier
				'eyebrow', // custom block fallback
				'text', // text block, cta
				'content', // legacy
				'alt', // image
				'quote' // quote block
			];
			for (const key of candidates) {
				const v = parsed[key];
				if (typeof v === 'string' && v.trim().length > 0) {
					// Strip the [[accent]] markers from the preview so the text
					// reads cleanly in the block list.
					const clean = v.replace(/\[\[|\]\]/g, '');
					return clean.length > 60 ? `${clean.slice(0, 60)}.` : clean;
				}
			}
			// Fall through: count items[] / tiers[] / paragraphs[] so the user
			// at least sees how many entries the block holds.
			for (const key of ['items', 'tiers', 'paragraphs']) {
				const arr = parsed[key];
				if (Array.isArray(arr) && arr.length > 0) {
					return `${arr.length} ${key}`;
				}
			}
		} catch {
			// ignore
		}
		return 'No content yet.';
	}

	const summaryText = $derived(summary());
</script>

<div
	role="listitem"
	draggable={!expanded}
	{ondragstart}
	{ondragover}
	{ondragend}
	{ondrop}
	class="rounded-xl border border-border/50 bg-bg-surface transition-colors {dragging
		? 'ring-1 ring-accent opacity-60'
		: 'hover:bg-bg-elevated/40'}"
>
	<div class="flex items-center gap-2 px-3 py-2.5">
		<button
			type="button"
			class="cursor-grab text-text-muted transition-colors hover:text-text-primary active:cursor-grabbing"
			aria-label="Drag to reorder"
			tabindex={-1}
		>
			<GripVertical class="h-4 w-4" />
		</button>
		<button
			type="button"
			class="flex flex-1 items-center gap-3 text-left focus-visible:outline-none"
			onclick={onToggleExpanded}
		>
			<span
				class="inline-flex h-5 items-center rounded-md bg-bg-elevated px-1.5 font-mono text-[10px] uppercase tracking-wider text-text-secondary"
			>
				{block.block_type}
			</span>
			<span class="truncate text-[13px] text-text-secondary">{summaryText}</span>
		</button>
		<div class="flex items-center gap-2">
			<Switch bind:checked={visibleBound} />
			<button
				type="button"
				aria-pressed={codeOpen}
				aria-label={codeOpen ? 'Hide source' : 'Show source'}
				title={codeOpen ? 'Hide source' : 'Show source'}
				class="inline-flex h-7 w-7 items-center justify-center rounded-md transition-colors {codeOpen
					? 'bg-bg-elevated text-text-primary'
					: 'text-text-muted hover:bg-bg-hover hover:text-text-primary'}"
				onclick={toggleCode}
			>
				<Code class="h-3.5 w-3.5" />
			</button>
			<button
				type="button"
				class="inline-flex h-7 items-center gap-1 rounded-md px-2 text-[12px] text-text-muted transition-colors hover:bg-bg-hover hover:text-danger"
				onclick={onDelete}
			>
				<Trash2 class="h-3.5 w-3.5" />
				Delete
			</button>
			<button
				type="button"
				class="inline-flex h-7 w-7 items-center justify-center rounded-md text-text-muted transition-colors hover:bg-bg-hover hover:text-text-primary"
				aria-label={expanded ? 'Collapse' : 'Expand'}
				onclick={onToggleExpanded}
			>
				<ChevronDown
					class="h-4 w-4 transition-transform {expanded ? 'rotate-180' : ''}"
				/>
			</button>
		</div>
	</div>
	{#if expanded}
		<div class="border-t border-border-light px-4 py-4">
			<BlockTypeForm
				blockType={block.block_type}
				dataJson={block.data_json || '{}'}
				onChange={onDataChange}
			/>
		</div>
	{/if}
	{#if codeOpen}
		<div class="border-t border-border-light bg-bg-elevated/40">
			<div class="flex items-center justify-between gap-2 px-4 py-2">
				<div class="flex items-center gap-2">
					<span class="text-[10.5px] font-mono uppercase tracking-[0.18em] text-text-muted">
						Source
					</span>
					{#if codeBlockType}
						<span class="font-mono text-[10.5px] text-text-secondary">{codeBlockType}.astro</span>
					{/if}
				</div>
				<button
					type="button"
					aria-label="Copy source"
					title="Copy"
					class="inline-flex h-6 w-6 items-center justify-center rounded text-text-muted hover:bg-bg-hover hover:text-text-primary disabled:opacity-50"
					disabled={!codeAstro}
					onclick={copyCode}
				>
					{#if copied}
						<Check class="h-3 w-3 text-accent" />
					{:else}
						<Copy class="h-3 w-3" />
					{/if}
				</button>
			</div>
			{#if codeLoading}
				<p class="px-4 pb-3 text-[12px] text-text-muted">Loading…</p>
			{:else if codeError}
				<p class="px-4 pb-3 text-[12px] text-danger">{codeError}</p>
			{:else if codeAstro !== null}
				<pre
					class="overflow-x-auto px-4 pb-3 font-mono text-[11.5px] leading-relaxed text-text-primary"
				>{codeAstro}</pre>
			{/if}
		</div>
	{/if}
</div>

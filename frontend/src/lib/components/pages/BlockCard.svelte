<script lang="ts">
	import { GripVertical, ChevronDown, Trash2 } from 'lucide-svelte';
	import Switch from '$lib/components/ui/Switch.svelte';
	import BlockTypeForm from './BlockTypeForm.svelte';
	import type { Block } from '$lib/api/types';

	let {
		block,
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
			const candidates = ['headline', 'title', 'content', 'alt'];
			for (const key of candidates) {
				const v = parsed[key];
				if (typeof v === 'string' && v.trim().length > 0) {
					return v.length > 60 ? `${v.slice(0, 60)}.` : v;
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
</div>

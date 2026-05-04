<script lang="ts">
	import BlockCard from './BlockCard.svelte';
	import BlockTypePicker from './BlockTypePicker.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import type { Block } from '$lib/api/types';

	type EditableBlock = Block & { _localId: string };

	let {
		siteID,
		pageID,
		initialBlocks,
		onChange
	}: {
		siteID: string;
		pageID: string;
		initialBlocks: Block[];
		onChange?: (state: { blocks: Block[]; dirty: boolean }) => void;
	} = $props();

	function clone(blocks: Block[]): EditableBlock[] {
		return blocks.map((b, i) => ({
			...b,
			name: b.name || '',
			data_json: b.data_json || '{}',
			style_json: b.style_json || '{}',
			sort_order: i,
			_localId: b.id || `tmp-${i}-${Math.random().toString(36).slice(2, 8)}`
		}));
	}

	let blocks = $state<EditableBlock[]>([]);
	let initialSnapshot = $state<EditableBlock[]>([]);
	let expandedId = $state<string | null>(null);
	let draggingId = $state<string | null>(null);
	let lastInitialRef: Block[] | null = null;

	$effect(() => {
		const incoming = initialBlocks;
		if (incoming !== lastInitialRef) {
			lastInitialRef = incoming;
			blocks = clone(incoming);
			initialSnapshot = clone(incoming);
			expandedId = null;
			draggingId = null;
		}
	});

	function serialize(list: EditableBlock[]): string {
		return JSON.stringify(
			list.map((b) => ({
				id: b.id || '',
				block_type: b.block_type,
				name: b.name || '',
				sort_order: b.sort_order,
				data_json: b.data_json || '{}',
				style_json: b.style_json || '{}',
				is_visible: b.is_visible
			}))
		);
	}

	const dirty = $derived(serialize(blocks) !== serialize(initialSnapshot));

	$effect(() => {
		const plain = blocks.map((b) => ({
			id: b.id,
			page_id: b.page_id,
			block_type: b.block_type,
			name: b.name || '',
			sort_order: b.sort_order,
			data_json: b.data_json,
			style_json: b.style_json,
			is_visible: b.is_visible,
			template_version: b.template_version,
			created_at: b.created_at,
			updated_at: b.updated_at
		}));
		onChange?.({ blocks: plain, dirty });
	});

	export function reset() {
		blocks = clone(initialSnapshot.map((b) => ({ ...b })));
		expandedId = null;
		draggingId = null;
	}

	export function getBlocks(): Block[] {
		return blocks.map((b) => ({
			id: b.id,
			page_id: b.page_id,
			block_type: b.block_type,
			name: b.name || '',
			sort_order: b.sort_order,
			data_json: b.data_json,
			style_json: b.style_json,
			is_visible: b.is_visible,
			template_version: b.template_version,
			created_at: b.created_at,
			updated_at: b.updated_at
		}));
	}

	export function syncFrom(next: Block[], preserveExpandedBlockId?: string) {
		// Preserve expansion across saves so the user's open block stays open.
		// We have to remap from server block.id -> local _localId because clone()
		// regenerates _localId for new blocks.
		const prevExpanded = expandedId;
		const prevExpandedBlockId =
			preserveExpandedBlockId ??
			blocks.find((b) => b._localId === prevExpanded)?.id ??
			'';
		blocks = clone(next);
		initialSnapshot = clone(next);
		const match = prevExpandedBlockId
			? blocks.find((b) => b.id === prevExpandedBlockId)
			: undefined;
		expandedId = match ? match._localId : null;
		draggingId = null;
	}

	function localIndex(localId: string): number {
		return blocks.findIndex((b) => b._localId === localId);
	}

	function handleDragStart(localId: string, e: DragEvent) {
		draggingId = localId;
		if (e.dataTransfer) {
			e.dataTransfer.effectAllowed = 'move';
			e.dataTransfer.setData('text/plain', localId);
		}
	}

	function handleDragOver(localId: string, e: DragEvent) {
		e.preventDefault();
		if (e.dataTransfer) {
			e.dataTransfer.dropEffect = 'move';
		}
		if (!draggingId || draggingId === localId) return;
		const fromIdx = localIndex(draggingId);
		const toIdx = localIndex(localId);
		if (fromIdx === -1 || toIdx === -1 || fromIdx === toIdx) return;
		const next = [...blocks];
		const moved = next.splice(fromIdx, 1)[0];
		if (!moved) return;
		next.splice(toIdx, 0, moved);
		blocks = next.map((b, i) => ({ ...b, sort_order: i }));
	}

	function handleDragEnd() {
		draggingId = null;
	}

	function handleDrop(e: DragEvent) {
		e.preventDefault();
		draggingId = null;
	}

	function toggleExpanded(localId: string) {
		expandedId = expandedId === localId ? null : localId;
	}

	function toggleVisibility(localId: string, visible: boolean) {
		const next = blocks.map((b) =>
			b._localId === localId ? { ...b, is_visible: (visible ? 1 : 0) as 0 | 1 } : b
		);
		blocks = next;
	}

	function setDataJson(localId: string, dataJson: string) {
		const next = blocks.map((b) => (b._localId === localId ? { ...b, data_json: dataJson } : b));
		blocks = next;
	}

	function setName(localId: string, name: string) {
		const next = blocks.map((b) => (b._localId === localId ? { ...b, name } : b));
		blocks = next;
	}

	function setType(localId: string, blockType: string, dataJson: string) {
		// Used by the per-block code editor when a typed block is converted to
		// raw_astro. Patches block_type and data_json in lockstep so the dirty
		// snapshot picks up both as a single change.
		const next = blocks.map((b) =>
			b._localId === localId ? { ...b, block_type: blockType, data_json: dataJson } : b
		);
		blocks = next;
	}

	function deleteBlock(localId: string) {
		blocks = blocks
			.filter((b) => b._localId !== localId)
			.map((b, i) => ({ ...b, sort_order: i }));
		if (expandedId === localId) expandedId = null;
	}

	function addBlock(type: string) {
		const localId = `new-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 6)}`;
		const created: EditableBlock = {
			id: '',
			page_id: blocks[0]?.page_id ?? '',
			block_type: type,
			name: '',
			sort_order: blocks.length,
			data_json: '{}',
			style_json: '{}',
			is_visible: 1,
			template_version: 1,
			created_at: '',
			updated_at: '',
			_localId: localId
		};
		blocks = [...blocks, created];
		expandedId = localId;
	}
</script>

{#if blocks.length === 0}
	<div class="rounded-xl border border-dashed border-border-light bg-bg-surface">
		<EmptyState
			title="No blocks yet"
			description="Add a hero, text, image, or feature grid to get this page started."
		>
			{#snippet action()}
				<BlockTypePicker onPick={addBlock} />
			{/snippet}
		</EmptyState>
	</div>
{:else}
	<div class="flex flex-col gap-2" role="list">
		{#each blocks as block (block._localId)}
			<BlockCard
				{block}
				{siteID}
				{pageID}
				expanded={expandedId === block._localId}
				dragging={draggingId === block._localId}
				onToggleExpanded={() => toggleExpanded(block._localId)}
				onToggleVisibility={(v) => toggleVisibility(block._localId, v)}
				onDataChange={(json) => setDataJson(block._localId, json)}
				onNameChange={(name) => setName(block._localId, name)}
				onTypeChange={(type, json) => setType(block._localId, type, json)}
				onDelete={() => deleteBlock(block._localId)}
				ondragstart={(e) => handleDragStart(block._localId, e)}
				ondragover={(e) => handleDragOver(block._localId, e)}
				ondragend={handleDragEnd}
				ondrop={handleDrop}
			/>
		{/each}
	</div>
	<div class="mt-3 flex justify-center">
		<BlockTypePicker onPick={addBlock} />
	</div>
{/if}

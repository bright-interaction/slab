<script lang="ts">
	import type { Block } from '$lib/api/types';

	let {
		blocks,
		onFieldChange
	}: {
		blocks: Block[];
		onFieldChange: (blockID: string, fieldKey: string, nextValue: string) => void;
	} = $props();

	// Canonical text-bearing keys across atomicsite block types. Order
	// determines display order inside each block. Ordered most-prominent
	// first so a writer reads top-to-bottom the way the page reads.
	const TEXT_KEYS: { key: string; label: string; multiline?: boolean }[] = [
		{ key: 'eyebrow', label: 'Eyebrow' },
		{ key: 'headline', label: 'Headline' },
		{ key: 'heading', label: 'Heading' },
		{ key: 'title', label: 'Title' },
		{ key: 'subheading', label: 'Subheading', multiline: true },
		{ key: 'subtitle', label: 'Subtitle', multiline: true },
		{ key: 'lead', label: 'Lead', multiline: true },
		{ key: 'text', label: 'Text', multiline: true },
		{ key: 'body', label: 'Body', multiline: true },
		{ key: 'content', label: 'Content', multiline: true },
		{ key: 'cta_text', label: 'CTA label' },
		{ key: 'cta_label', label: 'CTA label' },
		{ key: 'primary_label', label: 'Primary CTA label' },
		{ key: 'secondary_label', label: 'Secondary CTA label' },
		{ key: 'alt', label: 'Image alt' },
		{ key: 'caption', label: 'Caption' },
		{ key: 'quote', label: 'Quote', multiline: true },
		{ key: 'attribution', label: 'Attribution' }
	];

	interface Row {
		blockID: string;
		blockType: string;
		fieldKey: string;
		fieldLabel: string;
		multiline: boolean;
		value: string;
		empty: boolean;
	}

	function parseData(json: string | undefined): Record<string, unknown> {
		if (!json) return {};
		try {
			const v = JSON.parse(json);
			return v && typeof v === 'object' && !Array.isArray(v) ? (v as Record<string, unknown>) : {};
		} catch {
			return {};
		}
	}

	const rowsByBlock = $derived.by(() => {
		const out: { block: Block; rows: Row[] }[] = [];
		for (const block of blocks) {
			if (block.is_visible !== 1) continue;
			const data = parseData(block.data_json);
			const rows: Row[] = [];
			for (const def of TEXT_KEYS) {
				const raw = data[def.key];
				if (raw === undefined || raw === null) continue;
				if (typeof raw !== 'string') continue;
				rows.push({
					blockID: block.id,
					blockType: block.block_type,
					fieldKey: def.key,
					fieldLabel: def.label,
					multiline: def.multiline ?? false,
					value: raw,
					empty: raw.trim() === ''
				});
			}
			if (rows.length > 0) {
				out.push({ block, rows });
			}
		}
		return out;
	});

	function handleBlur(row: Row, e: FocusEvent) {
		const el = e.currentTarget as HTMLElement;
		const next = (el.innerText ?? '').replace(/ /g, ' ').trim();
		if (next === row.value.trim()) return;
		if (!row.blockID) return; // unsaved new block, skip until page saves first
		onFieldChange(row.blockID, row.fieldKey, next);
	}

	function handleKeydown(row: Row, e: KeyboardEvent) {
		if (e.key === 'Enter' && !row.multiline) {
			e.preventDefault();
			(e.currentTarget as HTMLElement).blur();
		}
		if (e.key === 'Escape') {
			(e.currentTarget as HTMLElement).innerText = row.value;
			(e.currentTarget as HTMLElement).blur();
		}
	}
</script>

<div class="flex flex-col gap-4">
	{#if rowsByBlock.length === 0}
		<p class="rounded-xl border border-dashed border-border-light bg-bg-surface px-5 py-6 text-center text-[12.5px] text-text-muted">
			No text fields on this page yet. Add a hero, heading, or text block to start writing.
		</p>
	{/if}
	{#each rowsByBlock as group (group.block.id || group.block.block_type)}
		<section class="rounded-xl border border-border-light bg-bg-surface">
			<header class="flex items-center justify-between gap-3 border-b border-border-light px-4 py-2">
				<span class="font-mono text-[10.5px] uppercase tracking-[0.18em] text-text-secondary">
					{group.block.block_type}
				</span>
				<span class="text-[10.5px] text-text-muted">
					{group.rows.length}
					{group.rows.length === 1 ? 'field' : 'fields'}
				</span>
			</header>
			<div class="divide-y divide-border-light">
				{#each group.rows as row (row.fieldKey)}
					<div class="flex flex-col gap-1 px-4 py-3">
						<span class="text-[11px] font-medium text-text-secondary">{row.fieldLabel}</span>
						<div
							role="textbox"
							tabindex="0"
							aria-label={row.fieldLabel}
							aria-multiline={row.multiline ? 'true' : 'false'}
							contenteditable="true"
							class="min-h-[1.6em] rounded-md px-2 py-1.5 text-[14px] leading-relaxed text-text-primary outline-none focus:bg-bg-elevated/60 focus:ring-1 focus:ring-accent {row.empty
								? 'text-text-muted italic'
								: ''}"
							onblur={(e) => handleBlur(row, e)}
							onkeydown={(e) => handleKeydown(row, e)}
						>{row.value || `Add ${row.fieldLabel.toLowerCase()}…`}</div>
					</div>
				{/each}
			</div>
		</section>
	{/each}
</div>

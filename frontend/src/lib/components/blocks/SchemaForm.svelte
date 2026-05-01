<script lang="ts">
	import { Plus, Trash2 } from 'lucide-svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Textarea from '$lib/components/ui/Textarea.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Checkbox from '$lib/components/ui/Checkbox.svelte';
	import Self from './SchemaForm.svelte';
	import type { BlockSchemaField } from '$lib/api/types';

	let {
		fields,
		value,
		onChange
	}: {
		fields: BlockSchemaField[];
		value: Record<string, unknown>;
		onChange: (next: Record<string, unknown>) => void;
	} = $props();

	function patch(key: string, next: unknown): void {
		onChange({ ...value, [key]: next });
	}

	function emptyForField(field: BlockSchemaField): unknown {
		switch (field.kind) {
			case 'array':
				return [];
			case 'object':
				return {};
			case 'bool':
				return false;
			case 'number':
				return 0;
			default:
				return '';
		}
	}

	function emptyObjectFromSchema(schema: BlockSchemaField[]): Record<string, unknown> {
		const out: Record<string, unknown> = {};
		for (const f of schema) out[f.key] = emptyForField(f);
		return out;
	}

	function isStringArrayItem(itemSchema: BlockSchemaField[] | undefined): boolean {
		return !!itemSchema && itemSchema.length === 1 && itemSchema[0]!.key === '';
	}

	function addItem(field: BlockSchemaField): void {
		const arr = Array.isArray(value[field.key]) ? (value[field.key] as unknown[]) : [];
		const itemSchema = field.item_schema || [];
		const item = isStringArrayItem(itemSchema)
			? emptyForField(itemSchema[0]!)
			: emptyObjectFromSchema(itemSchema);
		patch(field.key, [...arr, item]);
	}

	function removeItem(field: BlockSchemaField, index: number): void {
		const arr = Array.isArray(value[field.key]) ? (value[field.key] as unknown[]) : [];
		patch(
			field.key,
			arr.filter((_, i) => i !== index)
		);
	}

	function updateItem(field: BlockSchemaField, index: number, next: unknown): void {
		const arr = Array.isArray(value[field.key]) ? [...(value[field.key] as unknown[])] : [];
		arr[index] = next;
		patch(field.key, arr);
	}

	function asString(v: unknown): string {
		return typeof v === 'string' ? v : v == null ? '' : String(v);
	}

	function asBool(v: unknown): boolean {
		return v === true || v === 1 || v === '1' || v === 'true';
	}

	function asObj(v: unknown): Record<string, unknown> {
		return v && typeof v === 'object' && !Array.isArray(v) ? (v as Record<string, unknown>) : {};
	}

	function asArr(v: unknown): unknown[] {
		return Array.isArray(v) ? v : [];
	}
</script>

<div class="flex flex-col gap-3">
	{#each fields as field (field.key)}
		{#if field.kind === 'text' || field.kind === 'url' || field.kind === 'image_id'}
			<Input
				label={field.label}
				placeholder={field.placeholder}
				hint={field.help}
				value={asString(value[field.key])}
				oninput={(e: Event) =>
					patch(field.key, (e.currentTarget as HTMLInputElement).value)}
			/>
		{:else if field.kind === 'textarea' || field.kind === 'richtext'}
			<Textarea
				label={field.label}
				placeholder={field.placeholder}
				hint={field.help}
				rows={3}
				value={asString(value[field.key])}
				oninput={(e: Event) =>
					patch(field.key, (e.currentTarget as HTMLTextAreaElement).value)}
			/>
		{:else if field.kind === 'number'}
			<Input
				type="number"
				label={field.label}
				hint={field.help}
				value={asString(value[field.key])}
				oninput={(e: Event) => {
					const n = Number((e.currentTarget as HTMLInputElement).value);
					patch(field.key, Number.isFinite(n) ? n : 0);
				}}
			/>
		{:else if field.kind === 'select' && field.options}
			<div class="flex flex-col gap-1.5">
				<span class="text-[12px] font-medium text-text-secondary">{field.label}</span>
				<Select
					options={field.options}
					value={asString(value[field.key])}
					onchange={(v: string) => patch(field.key, v)}
				/>
				{#if field.help}
					<p class="text-[12px] text-text-muted">{field.help}</p>
				{/if}
			</div>
		{:else if field.kind === 'bool'}
			<Checkbox
				label={field.label}
				checked={asBool(value[field.key])}
				onCheckedChange={(c: boolean) => patch(field.key, c)}
			/>
		{:else if field.kind === 'array' && field.item_schema}
			{@const items = asArr(value[field.key])}
			{@const itemSchema = field.item_schema}
			{@const stringItem = isStringArrayItem(itemSchema)}
			<div class="flex flex-col gap-2 rounded-lg border border-border-light bg-bg-elevated/40 p-3">
				<div class="flex items-center justify-between">
					<span class="text-[12px] font-medium text-text-secondary">{field.label}</span>
					<button
						type="button"
						class="inline-flex h-7 items-center gap-1 rounded-md px-2 text-[12px] text-text-muted transition-colors hover:bg-bg-hover hover:text-text-primary"
						onclick={() => addItem(field)}
					>
						<Plus class="h-3.5 w-3.5" />
						Add
					</button>
				</div>
				{#if field.help}
					<p class="text-[12px] text-text-muted">{field.help}</p>
				{/if}
				{#if items.length === 0}
					<p class="text-[12px] italic text-text-muted">No items yet.</p>
				{:else}
					{#each items as item, idx (idx)}
						<div class="flex flex-col gap-2 rounded-md border border-border-light bg-bg-surface p-3">
							<div class="flex items-center justify-between">
								<span class="text-[11px] font-mono uppercase tracking-wider text-text-muted">
									#{idx + 1}
								</span>
								<button
									type="button"
									class="inline-flex h-6 w-6 items-center justify-center rounded text-text-muted hover:bg-bg-hover hover:text-danger"
									onclick={() => removeItem(field, idx)}
									aria-label="Remove item"
								>
									<Trash2 class="h-3 w-3" />
								</button>
							</div>
							{#if stringItem && itemSchema[0]!.kind === 'textarea'}
								<Textarea
									value={asString(item)}
									rows={2}
									placeholder={itemSchema[0]!.placeholder}
									oninput={(e: Event) =>
										updateItem(field, idx, (e.currentTarget as HTMLTextAreaElement).value)}
								/>
							{:else if stringItem}
								<Input
									value={asString(item)}
									placeholder={itemSchema[0]!.placeholder}
									oninput={(e: Event) =>
										updateItem(field, idx, (e.currentTarget as HTMLInputElement).value)}
								/>
							{:else}
								<Self
									fields={itemSchema}
									value={asObj(item)}
									onChange={(next: Record<string, unknown>) => updateItem(field, idx, next)}
								/>
							{/if}
						</div>
					{/each}
				{/if}
			</div>
		{:else if field.kind === 'object' && field.fields}
			<div class="flex flex-col gap-2 rounded-lg border border-border-light bg-bg-elevated/40 p-3">
				<span class="text-[12px] font-medium text-text-secondary">{field.label}</span>
				{#if field.help}
					<p class="text-[12px] text-text-muted">{field.help}</p>
				{/if}
				<Self
					fields={field.fields}
					value={asObj(value[field.key])}
					onChange={(next: Record<string, unknown>) => patch(field.key, next)}
				/>
			</div>
		{/if}
	{/each}
</div>

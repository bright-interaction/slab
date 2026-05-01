<script lang="ts">
	import { onMount } from 'svelte';
	import { AlertTriangle } from 'lucide-svelte';
	import SchemaForm from '$lib/components/blocks/SchemaForm.svelte';
	import JsonEditor from '$lib/components/blocks/JsonEditor.svelte';
	import * as blocksApi from '$lib/api/blocks';
	import type { BlockSchema } from '$lib/api/types';

	let {
		blockType,
		dataJson,
		onChange
	}: {
		blockType: string;
		dataJson: string;
		onChange: (next: string) => void;
	} = $props();

	type AnyData = Record<string, unknown>;

	function parse(input: string): AnyData {
		if (!input || input.trim() === '') return {};
		try {
			const parsed = JSON.parse(input);
			if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
				return parsed as AnyData;
			}
		} catch {
			/* fall through */
		}
		return {};
	}

	let schemas = $state<BlockSchema[]>([]);
	let schemasLoaded = $state(false);
	let schemasError = $state<string | null>(null);
	let lastDataJson = $state('');
	let data = $state<AnyData>({});
	let initialized = $state(false);

	$effect(() => {
		if (!initialized) {
			initialized = true;
			lastDataJson = dataJson;
			data = parse(dataJson);
			return;
		}
		if (dataJson !== lastDataJson) {
			lastDataJson = dataJson;
			data = parse(dataJson);
		}
	});

	const schema = $derived<BlockSchema | undefined>(schemas.find((s) => s.type === blockType));

	onMount(() => {
		blocksApi
			.schemas()
			.then((all) => {
				schemas = all;
				schemasLoaded = true;
			})
			.catch((err) => {
				schemasError = err instanceof Error ? err.message : 'Failed to load schemas';
				schemasLoaded = true;
			});
	});

	function commit(next: AnyData): void {
		data = next;
		const json = JSON.stringify(next);
		lastDataJson = json;
		onChange(json);
	}

	function commitRaw(json: string): void {
		lastDataJson = json;
		data = parse(json);
		onChange(json);
	}

	function isGlobalSlot(t: string): boolean {
		return t === 'header' || t === 'footer';
	}
</script>

{#if !schemasLoaded}
	<p class="text-[12px] text-text-muted">Loading editor…</p>
{:else if schemasError}
	<div class="rounded-md border border-danger/30 bg-danger/5 px-3 py-2 text-[12px] text-danger">
		{schemasError}
	</div>
	<JsonEditor value={dataJson} onChange={commitRaw} />
{:else if isGlobalSlot(blockType)}
	<div class="rounded-md border border-border-light bg-bg-elevated/40 px-3 py-2 text-[12px] text-text-muted">
		Global blocks ({blockType}) are edited from the Global blocks page so changes apply
		everywhere.
	</div>
	<JsonEditor value={dataJson} onChange={commitRaw} minHeight={6} />
{:else if schema}
	<SchemaForm fields={schema.fields} value={data} onChange={commit} />
{:else}
	<div class="mb-3 flex items-start gap-2 rounded-md border border-warning/30 bg-warning/5 px-3 py-2 text-[12px] text-warning">
		<AlertTriangle class="h-3.5 w-3.5 mt-0.5 shrink-0" />
		<span>
			No schema registered for <code class="font-mono">{blockType}</code>. Edit raw JSON below.
		</span>
	</div>
	<JsonEditor value={dataJson} onChange={commitRaw} />
{/if}

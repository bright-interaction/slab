<script lang="ts">
	import Input from '$lib/components/ui/Input.svelte';
	import Textarea from '$lib/components/ui/Textarea.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import { Plus, Trash2, AlertTriangle } from 'lucide-svelte';

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
			return {};
		} catch {
			return {};
		}
	}

	let data = $state<AnyData>({});
	let rawText = $state('');
	let lastSyncedJson = $state('');
	let initialized = $state(false);

	$effect(() => {
		const incoming = dataJson;
		if (!initialized) {
			initialized = true;
			lastSyncedJson = incoming;
			data = parse(incoming);
			rawText = incoming;
			return;
		}
		if (incoming !== lastSyncedJson) {
			lastSyncedJson = incoming;
			data = parse(incoming);
			rawText = incoming;
		}
	});

	function commit(next: AnyData) {
		data = next;
		const json = JSON.stringify(next);
		lastSyncedJson = json;
		onChange(json);
	}

	function setField(key: string, value: unknown) {
		commit({ ...data, [key]: value });
	}

	function asString(v: unknown): string {
		return typeof v === 'string' ? v : '';
	}

	function asAlignment(v: unknown): 'left' | 'center' {
		return v === 'center' ? 'center' : 'left';
	}

	const alignmentOptions = [
		{ value: 'left', label: 'Left' },
		{ value: 'center', label: 'Center' }
	];

	const ctaVariantOptions = [
		{ value: 'primary', label: 'Primary' },
		{ value: 'secondary', label: 'Secondary' }
	];

	type FeatureItem = { title?: string; body?: string; icon?: string };

	function getFeatures(): FeatureItem[] {
		const items = data.items;
		return Array.isArray(items) ? (items as FeatureItem[]) : [];
	}

	function addFeature() {
		const items = [...getFeatures(), { title: '', body: '', icon: '' }];
		commit({ ...data, items });
	}

	function updateFeature(idx: number, patch: Partial<FeatureItem>) {
		const items = getFeatures().map((it, i) => (i === idx ? { ...it, ...patch } : it));
		commit({ ...data, items });
	}

	function removeFeature(idx: number) {
		const items = getFeatures().filter((_, i) => i !== idx);
		commit({ ...data, items });
	}

	function commitRaw() {
		try {
			const parsed = JSON.parse(rawText);
			if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
				commit(parsed as AnyData);
				return;
			}
		} catch {
			// fall through, leave raw as is
		}
		lastSyncedJson = rawText;
		onChange(rawText);
	}

	// Local Select-bound values that sync back into data via $effect.
	let alignmentValue = $state<'left' | 'center'>('left');
	let ctaVariantValue = $state<'primary' | 'secondary'>('primary');

	$effect(() => {
		const nextAlignment = asAlignment(data.alignment);
		if (nextAlignment !== alignmentValue) {
			alignmentValue = nextAlignment;
		}
	});

	$effect(() => {
		const nextVariant = data.variant === 'secondary' ? 'secondary' : 'primary';
		if (nextVariant !== ctaVariantValue) {
			ctaVariantValue = nextVariant;
		}
	});

	function syncAlignment() {
		if (asAlignment(data.alignment) !== alignmentValue) {
			setField('alignment', alignmentValue);
		}
	}

	function syncCtaVariant() {
		const current = data.variant === 'secondary' ? 'secondary' : 'primary';
		if (current !== ctaVariantValue) {
			setField('variant', ctaVariantValue);
		}
	}

	$effect(syncAlignment);
	$effect(syncCtaVariant);
</script>

{#if blockType === 'hero'}
	<div class="grid grid-cols-1 gap-3 md:grid-cols-2">
		<Input
			label="Headline"
			value={asString(data.headline)}
			oninput={(e) => setField('headline', (e.currentTarget as HTMLInputElement).value)}
		/>
		<Input
			label="Subheadline"
			value={asString(data.subheadline)}
			oninput={(e) => setField('subheadline', (e.currentTarget as HTMLInputElement).value)}
		/>
		<Input
			label="CTA label"
			value={asString(data.cta_label)}
			oninput={(e) => setField('cta_label', (e.currentTarget as HTMLInputElement).value)}
		/>
		<Input
			label="CTA URL"
			value={asString(data.cta_href)}
			oninput={(e) => setField('cta_href', (e.currentTarget as HTMLInputElement).value)}
			placeholder="https://"
		/>
		<Input
			label="Image ID"
			value={asString(data.image_id)}
			oninput={(e) => setField('image_id', (e.currentTarget as HTMLInputElement).value)}
			hint="Media picker arrives in commit 11."
		/>
		<div class="flex flex-col gap-1.5">
			<span class="text-[12px] font-medium text-text-secondary">Alignment</span>
			<Select options={alignmentOptions} bind:value={alignmentValue} />
		</div>
	</div>
{:else if blockType === 'text'}
	<div class="flex flex-col gap-3">
		<Textarea
			label="Body"
			rows={6}
			value={asString(data.content)}
			oninput={(e) => setField('content', (e.currentTarget as HTMLTextAreaElement).value)}
		/>
		<div class="flex flex-col gap-1.5">
			<span class="text-[12px] font-medium text-text-secondary">Alignment</span>
			<Select options={alignmentOptions} bind:value={alignmentValue} />
		</div>
	</div>
{:else if blockType === 'image'}
	<div class="grid grid-cols-1 gap-3 md:grid-cols-2">
		<Input
			label="Image ID"
			value={asString(data.image_id)}
			oninput={(e) => setField('image_id', (e.currentTarget as HTMLInputElement).value)}
		/>
		<Input
			label="Alt text"
			value={asString(data.alt)}
			oninput={(e) => setField('alt', (e.currentTarget as HTMLInputElement).value)}
		/>
		<div class="md:col-span-2">
			<Input
				label="Caption"
				value={asString(data.caption)}
				oninput={(e) => setField('caption', (e.currentTarget as HTMLInputElement).value)}
			/>
		</div>
	</div>
{:else if blockType === 'cta'}
	<div class="grid grid-cols-1 gap-3 md:grid-cols-2">
		<Input
			label="Headline"
			value={asString(data.headline)}
			oninput={(e) => setField('headline', (e.currentTarget as HTMLInputElement).value)}
		/>
		<Input
			label="Button label"
			value={asString(data.primary_label)}
			oninput={(e) => setField('primary_label', (e.currentTarget as HTMLInputElement).value)}
		/>
		<div class="md:col-span-2">
			<Textarea
				label="Body"
				rows={3}
				value={asString(data.body)}
				oninput={(e) => setField('body', (e.currentTarget as HTMLTextAreaElement).value)}
			/>
		</div>
		<Input
			label="Button URL"
			value={asString(data.primary_href)}
			oninput={(e) => setField('primary_href', (e.currentTarget as HTMLInputElement).value)}
			placeholder="https://"
		/>
		<div class="flex flex-col gap-1.5">
			<span class="text-[12px] font-medium text-text-secondary">Variant</span>
			<Select options={ctaVariantOptions} bind:value={ctaVariantValue} />
		</div>
	</div>
{:else if blockType === 'feature_grid'}
	<div class="flex flex-col gap-3">
		<Input
			label="Headline"
			value={asString(data.headline)}
			oninput={(e) => setField('headline', (e.currentTarget as HTMLInputElement).value)}
		/>
		<Input
			label="Subheadline"
			value={asString(data.subheadline)}
			oninput={(e) => setField('subheadline', (e.currentTarget as HTMLInputElement).value)}
		/>
		<div class="flex flex-col gap-2">
			<div class="flex items-baseline justify-between">
				<p class="text-[12px] font-medium text-text-secondary">Features</p>
				<span class="text-[11px] text-text-muted">{getFeatures().length} item(s)</span>
			</div>
			{#each getFeatures() as item, idx (idx)}
				<div class="rounded-lg border border-border-light bg-bg p-3">
					<div class="grid grid-cols-1 gap-2 md:grid-cols-3">
						<Input
							label="Title"
							value={asString(item.title)}
							oninput={(e) =>
								updateFeature(idx, { title: (e.currentTarget as HTMLInputElement).value })}
						/>
						<Input
							label="Icon name"
							value={asString(item.icon)}
							oninput={(e) =>
								updateFeature(idx, { icon: (e.currentTarget as HTMLInputElement).value })}
							placeholder="lucide-name"
						/>
						<div class="flex items-end">
							<button
								type="button"
								class="inline-flex h-9 items-center justify-center gap-1.5 rounded-lg border border-border bg-bg-elevated px-3 text-[12px] text-text-muted transition-colors hover:bg-bg-hover hover:text-danger"
								onclick={() => removeFeature(idx)}
							>
								<Trash2 class="h-3.5 w-3.5" />
								Remove
							</button>
						</div>
					</div>
					<div class="mt-2">
						<Textarea
							label="Body"
							rows={2}
							value={asString(item.body)}
							oninput={(e) =>
								updateFeature(idx, { body: (e.currentTarget as HTMLTextAreaElement).value })}
						/>
					</div>
				</div>
			{/each}
			<div>
				<Button variant="ghost" size="sm" onclick={addFeature}>
					{#snippet icon()}
						<Plus class="h-3.5 w-3.5" />
					{/snippet}
					Add feature
				</Button>
			</div>
		</div>
	</div>
{:else if blockType === 'header' || blockType === 'footer'}
	<div class="rounded-lg border border-border-light bg-bg-elevated p-4">
		<p class="text-[12px] text-text-secondary">
			This is a global block. Edit it in
			<span class="font-medium text-text-primary">Settings</span> for the site.
		</p>
		<pre
			class="mt-3 overflow-x-auto rounded-md bg-bg p-3 font-mono text-[11px] text-text-muted">{JSON.stringify(
				data,
				null,
				2
			)}</pre>
	</div>
{:else}
	<div class="flex flex-col gap-2">
		<div
			class="flex items-start gap-2 rounded-lg border border-warning/30 bg-warning/10 px-3 py-2 text-[12px] text-warning"
		>
			<AlertTriangle class="h-3.5 w-3.5 shrink-0" />
			<span>
				Unknown block type "<span class="font-mono">{blockType}</span>". Editing raw JSON.
			</span>
		</div>
		<Textarea
			label="data_json"
			rows={6}
			value={rawText}
			oninput={(e) => {
				rawText = (e.currentTarget as HTMLTextAreaElement).value;
			}}
			onblur={commitRaw}
		/>
	</div>
{/if}

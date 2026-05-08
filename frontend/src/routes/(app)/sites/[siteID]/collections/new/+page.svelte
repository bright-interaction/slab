<script lang="ts">
	import { goto } from '$app/navigation';
	import { ArrowLeft, Plus, Trash2 } from 'lucide-svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Switch from '$lib/components/ui/Switch.svelte';
	import { ApiError } from '$lib/api/client';
	import * as collectionsApi from '$lib/api/collections';
	import type { FieldDef, FieldType } from '$lib/api/collections';
	import { toast } from '$lib/stores/toast.svelte';

	let { data }: { data: { site: { id: string } } } = $props();
	const siteID = $derived(data.site.id);

	let name = $state('');
	let slug = $state('');
	let schemaOrgType = $state('');
	let renderAsPages = $state(true);

	const fieldTypes: FieldType[] = [
		'text', 'textarea', 'richtext', 'number', 'boolean',
		'date', 'datetime', 'url', 'email',
		'image', 'gallery', 'select', 'multiselect',
		'reference', 'color', 'json'
	];
	const schemaOrgTypes = [
		'', 'Article', 'BlogPosting', 'NewsArticle', 'Product',
		'Person', 'Event', 'JobPosting', 'Recipe', 'FAQPage', 'Review'
	];

	let fields = $state<FieldDef[]>([
		{ name: 'description', type: 'textarea', label: 'Description', required: false }
	]);
	let saving = $state(false);

	function addField() {
		fields.push({ name: '', type: 'text', label: '', required: false });
	}

	function removeField(idx: number) {
		fields.splice(idx, 1);
	}

	function autoSlug() {
		if (!slug && name) {
			slug = name.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '').slice(0, 64);
		}
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		if (saving) return;
		const cleaned = fields
			.map((f) => ({ ...f, name: f.name.trim(), label: f.label.trim() || f.name.trim() }))
			.filter((f) => f.name);
		if (cleaned.length === 0) {
			toast.error('Add at least one field.');
			return;
		}
		saving = true;
		try {
			const settings: Record<string, unknown> = { render_as_pages: renderAsPages };
			if (schemaOrgType) settings.schema_org_type = schemaOrgType;
			const created = await collectionsApi.createCollection(siteID, {
				name: name.trim(),
				slug: slug.trim(),
				schema: cleaned,
				settings
			});
			toast.success('Collection created.');
			await goto(`/sites/${siteID}/collections/${created.id}`);
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Failed to create collection.');
		} finally {
			saving = false;
		}
	}
</script>

<div class="mx-auto max-w-3xl px-6 py-8">
	<a
		href={`/sites/${siteID}/collections`}
		class="inline-flex items-center gap-1.5 text-[12px] text-text-muted transition-colors hover:text-text-primary"
	>
		<ArrowLeft size={14} strokeWidth={1.75} />
		Collections
	</a>
	<header class="mt-3">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">New collection</h1>
		<p class="mt-1 text-[13px] text-text-secondary">
			Define the schema once. Items conform to it. Picking a schema.org type gets you auto JSON-LD.
		</p>
	</header>

	<form class="mt-8 flex flex-col gap-5" onsubmit={save}>
		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Identity</h2>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
				<Input label="Name" required bind:value={name} placeholder="Case studies" onblur={autoSlug} />
				<Input label="Slug" required bind:value={slug} placeholder="case-studies" hint="Lowercase, kebab-case. Drives the URL: /{slug}/" />
			</div>
		</Card>

		<Card padding="md">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Settings</h2>
			<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
				<Select
					label="Schema.org type"
					bind:value={schemaOrgType}
					options={schemaOrgTypes.map((t) => ({ value: t, label: t || '(none)' }))}
				/>
				<div class="flex flex-col gap-1.5">
					<span class="text-[12px] font-medium text-text-secondary">Render as pages</span>
					<div class="flex items-center gap-3 rounded-lg border border-border-light bg-bg-elevated px-3 py-2">
						<Switch bind:checked={renderAsPages} />
						<span class="text-[12px] text-text-muted">When on, items emit /{slug}/&lt;item-slug&gt;/ pages with auto JSON-LD.</span>
					</div>
				</div>
			</div>
		</Card>

		<Card padding="md">
			<div class="flex items-start justify-between">
				<div>
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Fields</h2>
					<p class="mt-1 text-[12px] text-text-muted">Field names should match schema.org conventions (headline, author, datePublished, etc.) for auto JSON-LD.</p>
				</div>
				<Button variant="secondary" size="sm" onclick={addField} type="button">
					<Plus size={12} strokeWidth={1.75} />Add field
				</Button>
			</div>
			<div class="mt-4 flex flex-col gap-3">
				{#each fields as f, idx (idx)}
					<div class="rounded-lg border border-border-light bg-bg-elevated p-3">
						<div class="grid grid-cols-1 gap-3 sm:grid-cols-12">
							<div class="sm:col-span-3">
								<Input label="Name" bind:value={f.name} placeholder="headline" />
							</div>
							<div class="sm:col-span-3">
								<Input label="Label" bind:value={f.label} placeholder="Headline" />
							</div>
							<div class="sm:col-span-3">
								<Select
									label="Type"
									bind:value={f.type}
									options={fieldTypes.map((t) => ({ value: t, label: t }))}
								/>
							</div>
							<div class="sm:col-span-2 flex items-end">
								<label class="flex items-center gap-2 pb-2 text-[12px] text-text-secondary">
									<input type="checkbox" bind:checked={f.required} />
									Required
								</label>
							</div>
							<div class="sm:col-span-1 flex items-end justify-end">
								<button
									type="button"
									class="inline-flex h-8 w-8 items-center justify-center rounded text-text-muted transition-colors hover:bg-bg-hover hover:text-danger"
									onclick={() => removeField(idx)}
									aria-label="Remove field"
								>
									<Trash2 size={14} strokeWidth={1.75} />
								</button>
							</div>
						</div>
					</div>
				{/each}
				{#if fields.length === 0}
					<p class="text-[12px] text-text-muted">No fields yet. Add at least one.</p>
				{/if}
			</div>
		</Card>

		<div class="flex items-center justify-end gap-2">
			<Button variant="secondary" onclick={() => goto(`/sites/${siteID}/collections`)} type="button">
				Cancel
			</Button>
			<Button variant="primary" type="submit" loading={saving} disabled={saving}>
				Create collection
			</Button>
		</div>
	</form>
</div>

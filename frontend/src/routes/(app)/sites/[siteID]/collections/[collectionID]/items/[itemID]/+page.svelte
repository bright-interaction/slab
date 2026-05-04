<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { ArrowLeft, Trash2 } from 'lucide-svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import { ApiError } from '$lib/api/client';
	import * as collectionsApi from '$lib/api/collections';
	import type { Collection, CollectionItem, FieldDef } from '$lib/api/collections';
	import { confirm } from '$lib/stores/confirm.svelte';
	import { toast } from '$lib/stores/toast.svelte';

	let { data }: { data: { site: { id: string } } } = $props();
	const siteID = $derived(data.site.id);
	const collectionID = $derived($page.params.collectionID as string);
	const itemID = $derived($page.params.itemID as string);

	let col = $state<Collection | null>(null);
	let item = $state<CollectionItem | null>(null);
	let title = $state('');
	let slug = $state('');
	let locale = $state('');
	let status = $state<'draft' | 'published' | 'scheduled'>('draft');
	let fieldData = $state<Record<string, unknown>>({});
	let saving = $state(false);
	let loading = $state(true);

	$effect(() => {
		void load();
	});

	async function load() {
		loading = true;
		try {
			[col, item] = await Promise.all([
				collectionsApi.getCollection(siteID, collectionID),
				collectionsApi.getItem(siteID, collectionID, itemID)
			]);
			title = item.title;
			slug = item.slug;
			locale = item.locale;
			status = item.status;
			try { fieldData = JSON.parse(item.data_json) || {}; } catch { fieldData = {}; }
			for (const f of col.schema) {
				if (!(f.name in fieldData)) fieldData[f.name] = defaultValue(f);
			}
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Failed to load item.');
		} finally {
			loading = false;
		}
	}

	function defaultValue(f: FieldDef): unknown {
		switch (f.type) {
			case 'boolean': return false;
			case 'number': return 0;
			case 'gallery': case 'multiselect': return [];
			default: return '';
		}
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		if (saving) return;
		saving = true;
		try {
			await collectionsApi.updateItem(siteID, collectionID, itemID, {
				title: title.trim(),
				slug: slug.trim(),
				data: fieldData,
				locale: locale.trim(),
				status
			});
			toast.success('Item saved.');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Save failed.');
		} finally {
			saving = false;
		}
	}

	async function deleteItem() {
		const ok = await confirm({
			title: `Delete "${title}"?`,
			message: 'This permanently removes the item.',
			confirmText: 'Delete',
			variant: 'danger'
		});
		if (!ok) return;
		try {
			await collectionsApi.deleteItem(siteID, collectionID, itemID);
			toast.success('Item deleted.');
			await goto(`/sites/${siteID}/collections/${collectionID}`);
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Delete failed.');
		}
	}
</script>

<div class="mx-auto max-w-3xl px-6 py-8">
	<a
		href={`/sites/${siteID}/collections/${collectionID}`}
		class="inline-flex items-center gap-1.5 text-[12px] text-text-muted transition-colors hover:text-text-primary"
	>
		<ArrowLeft size={14} strokeWidth={1.75} />
		Back to collection
	</a>
	<header class="mt-3 flex items-start justify-between gap-3">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">Edit item</h1>
		<Button variant="danger" size="sm" onclick={deleteItem}>
			<Trash2 size={12} strokeWidth={1.75} />Delete
		</Button>
	</header>

	{#if loading}
		<p class="mt-8 text-[12px] text-text-muted">Loading...</p>
	{:else if col && item}
		<form class="mt-8 flex flex-col gap-5" onsubmit={save}>
			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Identity</h2>
				<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
					<Input label="Title" required bind:value={title} />
					<Input label="Slug" required bind:value={slug} />
					<Input label="Locale" bind:value={locale} placeholder="(default site lang)" />
					<Select label="Status" bind:value={status}>
						<option value="draft">Draft</option>
						<option value="published">Published</option>
						<option value="scheduled">Scheduled</option>
					</Select>
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Fields</h2>
				<div class="mt-4 flex flex-col gap-4">
					{#each col.schema as f (f.name)}
						<div>
							{#if f.type === 'textarea' || f.type === 'richtext' || f.type === 'json'}
								<label class="flex flex-col gap-1.5">
									<span class="text-[12px] font-medium text-text-secondary">{f.label || f.name}{f.required ? ' *' : ''}</span>
									<textarea
										class="rounded-lg border border-border-light bg-bg-surface px-3 py-2 font-mono text-[12px] text-text-primary"
										rows={f.type === 'richtext' ? 8 : 4}
										bind:value={fieldData[f.name] as string}
										required={f.required}
									></textarea>
								</label>
							{:else if f.type === 'boolean'}
								<label class="flex items-center gap-2 text-[12px] text-text-secondary">
									<input type="checkbox" bind:checked={fieldData[f.name] as boolean} />
									{f.label || f.name}
								</label>
							{:else if f.type === 'select'}
								<Select label={f.label || f.name} bind:value={fieldData[f.name] as string} required={f.required}>
									<option value=""></option>
									{#each f.options ?? [] as opt (opt)}
										<option value={opt}>{opt}</option>
									{/each}
								</Select>
							{:else if f.type === 'number'}
								<Input label={f.label || f.name} type="number" bind:value={fieldData[f.name] as number} required={f.required} />
							{:else}
								<Input
									label={f.label || f.name}
									type={f.type === 'email' ? 'email' : f.type === 'url' ? 'url' : 'text'}
									bind:value={fieldData[f.name] as string}
									required={f.required}
									hint={f.type === 'image' ? 'Media id (24-char hex).' : ''}
								/>
							{/if}
						</div>
					{/each}
				</div>
			</Card>

			<div class="flex items-center justify-end gap-2">
				<Button variant="secondary" type="button" onclick={() => goto(`/sites/${siteID}/collections/${collectionID}`)}>
					Cancel
				</Button>
				<Button variant="primary" type="submit" loading={saving} disabled={saving}>
					Save
				</Button>
			</div>
		</form>
	{/if}
</div>

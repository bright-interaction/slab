<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { ArrowLeft, Plus, Trash2 } from 'lucide-svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import { ApiError } from '$lib/api/client';
	import * as collectionsApi from '$lib/api/collections';
	import type { Collection, CollectionItem } from '$lib/api/collections';
	import { confirm } from '$lib/stores/confirm.svelte';
	import { toast } from '$lib/stores/toast.svelte';

	let { data }: { data: { site: { id: string } } } = $props();
	const siteID = $derived(data.site.id);
	const collectionID = $derived($page.params.collectionID as string);

	let col = $state<Collection | null>(null);
	let items = $state<CollectionItem[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);

	async function load() {
		loading = true;
		loadError = null;
		try {
			[col, items] = await Promise.all([
				collectionsApi.getCollection(siteID, collectionID),
				collectionsApi.listItems(siteID, collectionID)
			]);
		} catch (err) {
			loadError = err instanceof ApiError ? err.message : 'Failed to load collection.';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	async function deleteItem(itemID: string, title: string) {
		const ok = await confirm({
			title: `Delete "${title}"?`,
			message: 'This permanently removes the item.',
			confirmText: 'Delete',
			variant: 'danger'
		});
		if (!ok) return;
		try {
			await collectionsApi.deleteItem(siteID, collectionID, itemID);
			items = items.filter((i) => i.id !== itemID);
			toast.success('Item deleted.');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Delete failed.');
		}
	}

	async function deleteCollection() {
		if (!col) return;
		const ok = await confirm({
			title: `Delete collection "${col.name}"?`,
			message: `This removes the collection AND all ${col.item_count} items. Cannot be undone.`,
			confirmText: 'Delete forever',
			variant: 'danger'
		});
		if (!ok) return;
		try {
			await collectionsApi.deleteCollection(siteID, collectionID);
			toast.success('Collection deleted.');
			await goto(`/sites/${siteID}/collections`);
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Delete failed.');
		}
	}
</script>

<div class="mx-auto max-w-6xl px-6 py-8">
	<a
		href={`/sites/${siteID}/collections`}
		class="inline-flex items-center gap-1.5 text-[12px] text-text-muted transition-colors hover:text-text-primary"
	>
		<ArrowLeft size={14} strokeWidth={1.75} />
		Collections
	</a>

	{#if loading}
		<p class="mt-8 text-[12px] text-text-muted">Loading...</p>
	{:else if loadError}
		<Card padding="md" class="mt-8 border-danger/30">
			<p class="text-[13px] text-danger">{loadError}</p>
		</Card>
	{:else if col}
		<header class="mt-3 flex items-start justify-between gap-4">
			<div>
				<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">{col.name}</h1>
				<p class="mt-1 text-[12px] text-text-muted">
					<span class="font-mono">/{col.slug}</span>
					{#if col.settings.schema_org_type}
						<span class="ml-2 rounded bg-bg-elevated px-1.5 py-0.5 text-[10px] uppercase tracking-wider">{col.settings.schema_org_type}</span>
					{/if}
				</p>
			</div>
			<div class="flex items-center gap-2">
				<Button variant="primary" onclick={() => goto(`/sites/${siteID}/collections/${collectionID}/items/new`)}>
					<Plus size={14} strokeWidth={1.75} />New item
				</Button>
				<Button variant="danger" size="sm" onclick={deleteCollection}>
					<Trash2 size={12} strokeWidth={1.75} />Delete
				</Button>
			</div>
		</header>

		<Card padding="md" class="mt-6">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Schema</h2>
			<div class="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2">
				{#each col.schema as f (f.name)}
					<div class="flex items-baseline justify-between rounded border border-border-light bg-bg-elevated px-3 py-2">
						<div>
							<span class="font-mono text-[12px] text-text-primary">{f.name}</span>
							<span class="ml-2 text-[11px] text-text-muted">{f.type}</span>
							{#if f.required}<span class="ml-2 text-[10px] text-accent">required</span>{/if}
						</div>
					</div>
				{/each}
			</div>
		</Card>

		<header class="mt-8 flex items-center justify-between">
			<h2 class="font-display text-xl font-extralight tracking-tight text-text-primary">
				Items <span class="text-text-muted text-[14px]">({items.length})</span>
			</h2>
		</header>

		<div class="mt-3 flex flex-col gap-2">
			{#if items.length === 0}
				<EmptyState
					title="No items yet"
					description="Add one manually or use the bulk import endpoint to populate from CSV / external source."
				/>
			{:else}
				{#each items as item (item.id)}
					<Card padding="sm">
						<div class="flex items-center justify-between gap-3">
							<div class="min-w-0 flex-1">
								<a
									class="text-[14px] font-medium text-text-primary hover:underline"
									href={`/sites/${siteID}/collections/${collectionID}/items/${item.id}`}
								>
									{item.title}
								</a>
								<p class="mt-0.5 text-[11px] text-text-muted">
									<span class="font-mono">{item.slug}</span>
									{#if item.locale}
										<span class="ml-2">[{item.locale}]</span>
									{/if}
									<span class="ml-2 rounded bg-bg-elevated px-1.5 py-0.5 text-[10px] uppercase tracking-wider">{item.status}</span>
								</p>
							</div>
							<button
								type="button"
								class="inline-flex h-7 w-7 items-center justify-center rounded text-text-muted transition-colors hover:bg-bg-hover hover:text-danger"
								onclick={() => deleteItem(item.id, item.title)}
								aria-label="Delete item"
							>
								<Trash2 size={12} strokeWidth={1.75} />
							</button>
						</div>
					</Card>
				{/each}
			{/if}
		</div>
	{/if}
</div>

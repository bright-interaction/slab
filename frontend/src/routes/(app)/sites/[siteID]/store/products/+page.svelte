<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { page as pageState } from '$app/state';
	import * as productsApi from '$lib/api/products';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { ApiError } from '$lib/api/client';
	import type { Product, ProductStatus } from '$lib/api/products';

	const siteID = $derived(pageState.params.siteID as string);

	let products = $state<Product[]>([]);
	let loading = $state(true);
	let error = $state('');

	let createOpen = $state(false);
	let creating = $state(false);
	let newName = $state('');
	let newSlug = $state('');
	let newPriceEur = $state('0.00');
	let newCurrency = $state('EUR');
	let newStatus = $state<ProductStatus>('draft');

	const statusOptions = [
		{ value: 'draft', label: 'Draft' },
		{ value: 'active', label: 'Active' },
		{ value: 'archived', label: 'Archived' }
	];

	const currencyOptions = [
		{ value: 'EUR', label: 'EUR' },
		{ value: 'SEK', label: 'SEK' },
		{ value: 'USD', label: 'USD' },
		{ value: 'GBP', label: 'GBP' }
	];

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			products = await productsApi.listProducts(siteID);
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Failed to load products';
			products = [];
		} finally {
			loading = false;
		}
	}

	function slugify(input: string): string {
		return input
			.toLowerCase()
			.replace(/[^a-z0-9]+/g, '-')
			.replace(/^-+|-+$/g, '')
			.slice(0, 60);
	}

	$effect(() => {
		if (createOpen) {
			newSlug = slugify(newName);
		}
	});

	function priceCentsFromEur(eur: string): number {
		const n = Number(eur);
		if (!Number.isFinite(n) || n < 0) return 0;
		return Math.round(n * 100);
	}

	async function createProduct() {
		const name = newName.trim();
		const slug = newSlug.trim().toLowerCase();
		if (name.length < 2) {
			toast.error('Name must be at least 2 characters.');
			return;
		}
		if (!/^[a-z0-9][-a-z0-9_]{0,79}$/.test(slug)) {
			toast.error('Slug must be lowercase alphanumeric / dash / underscore.');
			return;
		}
		creating = true;
		try {
			const p = await productsApi.createProduct(siteID, {
				name,
				slug,
				status: newStatus,
				currency: newCurrency,
				base_price_cents: priceCentsFromEur(newPriceEur)
			});
			toast.success(`Created ${p.name}`);
			createOpen = false;
			newName = '';
			newSlug = '';
			newPriceEur = '0.00';
			newStatus = 'draft';
			await goto(`/sites/${siteID}/store/products/${p.id}`);
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Create failed');
		} finally {
			creating = false;
		}
	}

	function statusVariant(s: ProductStatus): 'success' | 'warning' | 'default' {
		if (s === 'active') return 'success';
		if (s === 'draft') return 'warning';
		return 'default';
	}

	function formatPrice(cents: number, currency: string): string {
		const sym = currency === 'EUR' ? '€' : currency === 'GBP' ? '£' : currency === 'USD' ? '$' : currency + ' ';
		return sym + (cents / 100).toFixed(2);
	}
</script>

<div class="space-y-4">
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-sm font-medium text-text-primary">Products</h2>
			<p class="text-xs text-text-muted mt-1">
				{products.length} product{products.length === 1 ? '' : 's'} in the catalog.
			</p>
		</div>
		<Button onclick={() => (createOpen = true)}>+ New product</Button>
	</div>

	<Card>
		{#if loading}
			<div class="space-y-2">
				<Skeleton class="h-10" />
				<Skeleton class="h-10" />
				<Skeleton class="h-10" />
			</div>
		{:else if error}
			<div class="rounded-lg border border-danger/30 bg-danger/5 p-3 text-xs text-danger">
				{error}
			</div>
		{:else if products.length === 0}
			<EmptyState
				title="No products yet"
				description="Create your first product. Each can have variants (sizes, colours, etc), per-variant inventory, and per-product pricing."
			>
				{#snippet action()}
					<Button onclick={() => (createOpen = true)}>+ New product</Button>
				{/snippet}
			</EmptyState>
		{:else}
			<div class="overflow-x-auto rounded-lg border border-border-light">
				<table class="w-full text-xs">
					<thead class="bg-bg-elevated text-text-muted">
						<tr>
							<th class="text-left font-normal px-3 py-2">Name</th>
							<th class="text-left font-normal px-3 py-2">Slug</th>
							<th class="text-left font-normal px-3 py-2">Status</th>
							<th class="text-right font-normal px-3 py-2 tabular-nums">Price</th>
							<th class="text-left font-normal px-3 py-2">Category</th>
							<th class="text-right font-normal px-3 py-2"></th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border-light">
						{#each products as p (p.id)}
							<tr class="hover:bg-bg-hover transition-colors">
								<td class="px-3 py-1.5 text-text-primary">{p.name}</td>
								<td class="px-3 py-1.5 font-mono text-text-muted truncate max-w-[20ch]">/{p.slug}</td>
								<td class="px-3 py-1.5">
									<Badge variant={statusVariant(p.status)}>{p.status}</Badge>
								</td>
								<td class="px-3 py-1.5 tabular-nums text-right">{formatPrice(p.base_price_cents, p.currency)}</td>
								<td class="px-3 py-1.5 text-text-muted">{p.category || '-'}</td>
								<td class="px-3 py-1.5 text-right">
									<a
										href={`/sites/${siteID}/store/products/${p.id}`}
										class="text-accent hover:underline"
									>Open →</a>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</Card>
</div>

<Dialog
	bind:open={createOpen}
	title="New product"
	description="Scaffold a product, then add variants and inventory on the detail page."
>
	<div class="space-y-3">
		<Input label="Name" bind:value={newName} placeholder="Premium T-Shirt" />
		<Input label="Slug" bind:value={newSlug} placeholder="premium-t-shirt" />
		<div class="grid grid-cols-2 gap-3">
			<Select label="Status" bind:value={newStatus} options={statusOptions} />
			<Select label="Currency" bind:value={newCurrency} options={currencyOptions} />
		</div>
		<Input
			label="Base price (incl. VAT)"
			bind:value={newPriceEur}
			placeholder="29.95"
			hint="In the selected currency. Per-variant prices can override this."
		/>
	</div>
	{#snippet footer()}
		<div class="flex justify-end gap-2">
			<Button variant="ghost" onclick={() => (createOpen = false)} disabled={creating}>
				Cancel
			</Button>
			<Button onclick={createProduct} loading={creating} disabled={newName.trim().length < 2}>
				Create
			</Button>
		</div>
	{/snippet}
</Dialog>

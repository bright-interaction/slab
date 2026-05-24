<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { page as pageState } from '$app/state';
	import * as productsApi from '$lib/api/products';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import { confirm } from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { ApiError } from '$lib/api/client';
	import type { Product, ProductStatus, ProductVariant, InventoryAdjustment, InventoryReason } from '$lib/api/products';

	const siteID = $derived(pageState.params.siteID as string);
	const productID = $derived(pageState.params.productID as string);

	let product = $state<Product | null>(null);
	let variants = $state<ProductVariant[]>([]);
	let loading = $state(true);
	let error = $state('');

	let dirty = $state(false);
	let saving = $state(false);

	let invOpen = $state(false);
	let invVariant = $state<ProductVariant | null>(null);
	let invNewCount = $state('');
	let invDelta = $state('');
	let invMode = $state<'absolute' | 'delta'>('absolute');
	let invReason = $state<InventoryReason>('manual');
	let invNote = $state('');
	let invSubmitting = $state(false);
	let invAdjustments = $state<InventoryAdjustment[]>([]);

	let addVariantOpen = $state(false);
	let newVariantName = $state('');
	let newVariantSku = $state('');
	let newVariantPriceEur = $state('0.00');
	let newVariantInventory = $state('0');
	let creatingVariant = $state(false);

	const statusOptions = [
		{ value: 'draft', label: 'Draft' },
		{ value: 'active', label: 'Active' },
		{ value: 'archived', label: 'Archived' }
	];
	const reasonOptions: { value: InventoryReason; label: string }[] = [
		{ value: 'restock', label: 'Restock' },
		{ value: 'sale', label: 'Sale' },
		{ value: 'manual', label: 'Manual' },
		{ value: 'damage', label: 'Damage' },
		{ value: 'return', label: 'Return' },
		{ value: 'correction', label: 'Correction' }
	];

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			product = await productsApi.getProduct(siteID, productID);
			variants = await productsApi.listVariants(siteID, productID);
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Failed to load product';
			product = null;
			variants = [];
		} finally {
			loading = false;
		}
	}

	function formatPrice(cents: number, currency: string): string {
		const sym = currency === 'EUR' ? '€' : currency === 'GBP' ? '£' : currency === 'USD' ? '$' : currency + ' ';
		return sym + (cents / 100).toFixed(2);
	}

	function priceCentsFromInput(eur: string): number {
		const n = Number(eur);
		if (!Number.isFinite(n) || n < 0) return 0;
		return Math.round(n * 100);
	}

	function markDirty() {
		dirty = true;
	}

	async function save() {
		if (!product) return;
		saving = true;
		try {
			const updated = await productsApi.updateProduct(siteID, product.id, {
				name: product.name,
				slug: product.slug,
				description: product.description,
				category: product.category,
				status: product.status,
				base_price_cents: product.base_price_cents,
				currency: product.currency,
				requires_shipping: product.requires_shipping === 1,
				tax_class: product.tax_class,
				weight_grams: product.weight_grams
			});
			product = updated;
			dirty = false;
			toast.success('Saved');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Save failed');
		} finally {
			saving = false;
		}
	}

	async function deleteProduct() {
		if (!product) return;
		const ok = await confirm({
			title: `Delete "${product.name}"?`,
			message: 'This deletes the product and all of its variants. Inventory audit logs are kept.',
			confirmText: 'Delete',
			variant: 'danger'
		});
		if (!ok) return;
		try {
			await productsApi.deleteProduct(siteID, product.id);
			toast.success('Product deleted');
			void goto(`/sites/${siteID}/store/products`);
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Delete failed');
		}
	}

	function openInventory(v: ProductVariant) {
		invVariant = v;
		invMode = 'absolute';
		invNewCount = String(v.inventory_count);
		invDelta = '';
		invReason = 'manual';
		invNote = '';
		invAdjustments = [];
		invOpen = true;
		void loadAdjustments(v.id);
	}

	async function loadAdjustments(variantID: string) {
		try {
			invAdjustments = await productsApi.listInventoryAdjustments(siteID, variantID);
		} catch {
			invAdjustments = [];
		}
	}

	async function submitInventory() {
		if (!invVariant) return;
		const body: { new_count?: number; delta?: number; reason: InventoryReason; note: string } = {
			reason: invReason,
			note: invNote.trim()
		};
		if (invMode === 'absolute') {
			const n = Number(invNewCount);
			if (!Number.isFinite(n)) {
				toast.error('Enter a number for new count.');
				return;
			}
			body.new_count = Math.round(n);
		} else {
			const d = Number(invDelta);
			if (!Number.isFinite(d) || d === 0) {
				toast.error('Enter a non-zero delta.');
				return;
			}
			body.delta = Math.round(d);
		}
		invSubmitting = true;
		try {
			const result = await productsApi.setInventory(siteID, invVariant.id, body);
			toast.success(`Inventory updated (delta ${result.delta >= 0 ? '+' : ''}${result.delta})`);
			variants = variants.map((v) => (v.id === result.variant.id ? result.variant : v));
			await loadAdjustments(invVariant.id);
			invMode = 'absolute';
			invNewCount = String(result.variant.inventory_count);
			invDelta = '';
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Inventory update failed');
		} finally {
			invSubmitting = false;
		}
	}

	async function createVariant() {
		const name = newVariantName.trim();
		if (name.length < 1) {
			toast.error('Variant needs a name (e.g. "Small / Black").');
			return;
		}
		creatingVariant = true;
		try {
			const created = await productsApi.createVariant(siteID, productID, {
				name,
				sku: newVariantSku.trim(),
				price_cents: priceCentsFromInput(newVariantPriceEur),
				inventory_count: Math.max(0, Math.round(Number(newVariantInventory) || 0))
			});
			variants = [...variants, created];
			toast.success(`Variant "${created.name}" added`);
			addVariantOpen = false;
			newVariantName = '';
			newVariantSku = '';
			newVariantPriceEur = '0.00';
			newVariantInventory = '0';
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Add variant failed');
		} finally {
			creatingVariant = false;
		}
	}

	async function deleteVariant(v: ProductVariant) {
		const ok = await confirm({
			title: `Delete variant "${v.name}"?`,
			message: 'This deletes the variant. Inventory audit logs are kept.',
			confirmText: 'Delete',
			variant: 'danger'
		});
		if (!ok) return;
		try {
			await productsApi.deleteVariant(siteID, v.id);
			variants = variants.filter((x) => x.id !== v.id);
			toast.success('Variant deleted');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Delete variant failed');
		}
	}
</script>

{#if loading}
	<div class="space-y-2">
		<Skeleton class="h-10" />
		<Skeleton class="h-32" />
	</div>
{:else if error || !product}
	<div class="rounded-lg border border-danger/30 bg-danger/5 p-3 text-xs text-danger">
		{error || 'Product not found'}
	</div>
{:else}
	<div class="space-y-6">
		<div class="flex items-center justify-between">
			<div>
				<a href={`/sites/${siteID}/store/products`} class="text-xs text-text-muted hover:text-text-primary">
					← Back to products
				</a>
				<h2 class="mt-1 text-lg font-medium text-text-primary">{product.name}</h2>
			</div>
			<div class="flex items-center gap-2">
				<Button variant="ghost" onclick={deleteProduct}>Delete</Button>
				<Button onclick={save} loading={saving} disabled={!dirty}>Save</Button>
			</div>
		</div>

		<Card>
			<h3 class="text-sm font-medium text-text-primary mb-3">Details</h3>
			<div class="grid grid-cols-2 gap-3">
				<Input
					label="Name"
					value={product.name}
					oninput={(e) => {
						product = { ...product!, name: (e.currentTarget as HTMLInputElement).value };
						markDirty();
					}}
				/>
				<Input
					label="Slug"
					value={product.slug}
					oninput={(e) => {
						product = { ...product!, slug: (e.currentTarget as HTMLInputElement).value };
						markDirty();
					}}
				/>
				<Select
					label="Status"
					value={product.status}
					options={statusOptions}
					onchange={(v) => {
						product = { ...product!, status: v as ProductStatus };
						markDirty();
					}}
				/>
				<Input
					label="Category"
					value={product.category}
					oninput={(e) => {
						product = { ...product!, category: (e.currentTarget as HTMLInputElement).value };
						markDirty();
					}}
					placeholder="apparel"
				/>
				<Input
					label={`Base price (${product.currency})`}
					value={(product.base_price_cents / 100).toFixed(2)}
					oninput={(e) => {
						product = {
							...product!,
							base_price_cents: priceCentsFromInput((e.currentTarget as HTMLInputElement).value)
						};
						markDirty();
					}}
				/>
				<Input
					label="Weight (grams)"
					value={String(product.weight_grams)}
					oninput={(e) => {
						product = {
							...product!,
							weight_grams: Math.max(0, Math.round(Number((e.currentTarget as HTMLInputElement).value) || 0))
						};
						markDirty();
					}}
				/>
			</div>
		</Card>

		<Card>
			<div class="flex items-center justify-between mb-3">
				<h3 class="text-sm font-medium text-text-primary">Variants</h3>
				<Button variant="secondary" onclick={() => (addVariantOpen = true)}>+ Add variant</Button>
			</div>
			{#if variants.length === 0}
				<EmptyState
					title="No variants"
					description="Add a variant to set per-SKU price + inventory. Even single-SKU products need one variant for the cart to work in Slice B."
				/>
			{:else}
				<div class="overflow-x-auto rounded-lg border border-border-light">
					<table class="w-full text-xs">
						<thead class="bg-bg-elevated text-text-muted">
							<tr>
								<th class="text-left font-normal px-3 py-2">Name</th>
								<th class="text-left font-normal px-3 py-2">SKU</th>
								<th class="text-right font-normal px-3 py-2 tabular-nums">Price</th>
								<th class="text-right font-normal px-3 py-2 tabular-nums">Inventory</th>
								<th class="text-right font-normal px-3 py-2"></th>
							</tr>
						</thead>
						<tbody class="divide-y divide-border-light">
							{#each variants as v (v.id)}
								<tr>
									<td class="px-3 py-1.5">{v.name || '(unnamed)'}</td>
									<td class="px-3 py-1.5 font-mono text-text-muted">{v.sku || '-'}</td>
									<td class="px-3 py-1.5 text-right tabular-nums">
										{formatPrice(v.price_cents || product.base_price_cents, product.currency)}
									</td>
									<td class="px-3 py-1.5 text-right tabular-nums">
										<Badge variant={v.inventory_count > 0 ? 'default' : 'warning'}>{v.inventory_count}</Badge>
									</td>
									<td class="px-3 py-1.5 text-right">
										<button
											type="button"
											class="text-accent text-xs hover:underline mr-3"
											onclick={() => openInventory(v)}
										>Inventory</button>
										<button
											type="button"
											class="text-text-muted text-xs hover:text-danger"
											onclick={() => deleteVariant(v)}
										>Delete</button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{/if}
		</Card>
	</div>
{/if}

<Dialog
	bind:open={addVariantOpen}
	title="Add variant"
	description="Variants carry per-SKU price + inventory."
>
	<div class="space-y-3">
		<Input label="Name" bind:value={newVariantName} placeholder="Small / Black" />
		<Input label="SKU" bind:value={newVariantSku} placeholder="SHIRT-S-BLK" />
		<div class="grid grid-cols-2 gap-3">
			<Input label="Price" bind:value={newVariantPriceEur} placeholder="29.95" hint="0 = use product base price" />
			<Input label="Inventory" bind:value={newVariantInventory} placeholder="0" />
		</div>
	</div>
	{#snippet footer()}
		<div class="flex justify-end gap-2">
			<Button variant="ghost" onclick={() => (addVariantOpen = false)} disabled={creatingVariant}>Cancel</Button>
			<Button onclick={createVariant} loading={creatingVariant}>Add</Button>
		</div>
	{/snippet}
</Dialog>

<Dialog
	bind:open={invOpen}
	size="lg"
	title="Inventory"
	description={invVariant ? `${invVariant.name || invVariant.sku || 'Variant'}, current count: ${invVariant.inventory_count}` : ''}
>
	<div class="space-y-4">
		<div class="flex items-center gap-2 text-xs">
			<button
				type="button"
				class="rounded px-2 py-1 {invMode === 'absolute' ? 'bg-bg-elevated font-medium text-text-primary' : 'text-text-muted hover:text-text-primary'}"
				onclick={() => (invMode = 'absolute')}
			>Set absolute</button>
			<button
				type="button"
				class="rounded px-2 py-1 {invMode === 'delta' ? 'bg-bg-elevated font-medium text-text-primary' : 'text-text-muted hover:text-text-primary'}"
				onclick={() => (invMode = 'delta')}
			>Apply delta</button>
		</div>
		{#if invMode === 'absolute'}
			<Input label="New count" bind:value={invNewCount} placeholder="0" />
		{:else}
			<Input label="Delta (signed)" bind:value={invDelta} placeholder="+50 or -3" />
		{/if}
		<Select label="Reason" bind:value={invReason} options={reasonOptions} />
		<Input label="Note (optional)" bind:value={invNote} placeholder="Restock from supplier ABC" />

		{#if invAdjustments.length > 0}
			<div>
				<div class="text-xs font-medium text-text-secondary mb-1.5">Recent adjustments</div>
				<div class="rounded-lg border border-border-light max-h-56 overflow-y-auto">
					<table class="w-full text-[11px]">
						<thead class="bg-bg-elevated text-text-muted">
							<tr>
								<th class="text-left font-normal px-2 py-1">When</th>
								<th class="text-left font-normal px-2 py-1">Reason</th>
								<th class="text-right font-normal px-2 py-1 tabular-nums">Delta</th>
								<th class="text-right font-normal px-2 py-1 tabular-nums">New count</th>
								<th class="text-left font-normal px-2 py-1">Note</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-border-light">
							{#each invAdjustments as a (a.id)}
								<tr>
									<td class="px-2 py-1 text-text-muted">{a.created_at}</td>
									<td class="px-2 py-1">{a.reason}</td>
									<td class="px-2 py-1 text-right tabular-nums {a.delta < 0 ? 'text-danger' : ''}">
										{a.delta >= 0 ? '+' : ''}{a.delta}
									</td>
									<td class="px-2 py-1 text-right tabular-nums">{a.new_count}</td>
									<td class="px-2 py-1 text-text-muted truncate max-w-[24ch]">{a.note}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{/if}
	</div>
	{#snippet footer()}
		<div class="flex justify-end gap-2">
			<Button variant="ghost" onclick={() => (invOpen = false)} disabled={invSubmitting}>Close</Button>
			<Button onclick={submitInventory} loading={invSubmitting}>Apply</Button>
		</div>
	{/snippet}
</Dialog>

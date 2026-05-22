<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { page as pageState } from '$app/state';
	import * as ordersApi from '$lib/api/orders';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import { confirm } from '$lib/components/ui/ConfirmDialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { ApiError } from '$lib/api/client';
	import type { OrderDetail, OrderStatus } from '$lib/api/orders';

	const siteID = $derived(pageState.params.siteID as string);
	const orderID = $derived(pageState.params.orderID as string);

	let data = $state<OrderDetail | null>(null);
	let loading = $state(true);
	let error = $state('');
	let acting = $state<'fulfill' | 'cancel' | 'refund' | null>(null);
	let notes = $state('');
	let savingNotes = $state(false);

	const transitions: Record<OrderStatus, OrderStatus[]> = {
		pending: ['paid', 'cancelled', 'failed'],
		paid: ['fulfilled', 'refunded'],
		fulfilled: ['refunded'],
		cancelled: [],
		refunded: [],
		failed: []
	};

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			data = await ordersApi.getOrder(siteID, orderID);
			notes = data.order.notes;
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Failed to load order';
			data = null;
		} finally {
			loading = false;
		}
	}

	function formatPrice(cents: number, currency: string): string {
		const sym = currency === 'EUR' ? '€' : currency === 'GBP' ? '£' : currency === 'USD' ? '$' : currency + ' ';
		return sym + (cents / 100).toFixed(2);
	}

	function statusVariant(s: OrderStatus): 'success' | 'warning' | 'danger' | 'default' {
		if (s === 'paid' || s === 'fulfilled') return 'success';
		if (s === 'pending') return 'warning';
		if (s === 'cancelled' || s === 'failed' || s === 'refunded') return 'danger';
		return 'default';
	}

	async function setStatus(next: OrderStatus, actionKey: 'fulfill' | 'cancel') {
		if (!data) return;
		const ok = await confirm({
			title: `Mark order ${next}?`,
			message: `The order will transition from "${data.order.status}" to "${next}". This is reversible only via further transitions allowed by the state machine.`,
			confirmText: 'Confirm',
			variant: actionKey === 'cancel' ? 'danger' : 'primary'
		});
		if (!ok) return;
		acting = actionKey;
		try {
			data = await ordersApi.updateOrderStatus(siteID, orderID, { status: next });
			toast.success(`Order ${next}`);
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Status update failed');
		} finally {
			acting = null;
		}
	}

	async function refund() {
		if (!data) return;
		const ok = await confirm({
			title: 'Refund the full order?',
			message: 'This triggers a Mollie refund for the full amount and flips the order to refunded. The Mollie refund itself can take minutes to confirm via webhook.',
			confirmText: 'Refund',
			variant: 'danger'
		});
		if (!ok) return;
		acting = 'refund';
		try {
			data = await ordersApi.refundOrder(siteID, orderID);
			toast.success('Refund initiated at Mollie');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Refund failed');
		} finally {
			acting = null;
		}
	}

	async function saveNotes() {
		if (!data) return;
		savingNotes = true;
		try {
			data = await ordersApi.updateOrderStatus(siteID, orderID, {
				status: data.order.status,
				notes
			});
			toast.success('Notes saved');
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Save notes failed');
		} finally {
			savingNotes = false;
		}
	}
</script>

{#if loading}
	<div class="space-y-3"><Skeleton class="h-10" /><Skeleton class="h-40" /></div>
{:else if error || !data}
	<div class="rounded-lg border border-danger/30 bg-danger/5 p-3 text-xs text-danger">{error || 'Order not found'}</div>
{:else}
	{@const o = data.order}
	{@const allowed = transitions[o.status]}
	<div class="space-y-6">
		<div class="flex items-center justify-between">
			<div>
				<a href={`/sites/${siteID}/store/orders`} class="text-xs text-text-muted hover:text-text-primary">
					← Back to orders
				</a>
				<h2 class="mt-1 text-lg font-medium text-text-primary">
					{o.order_number}
					<Badge variant={statusVariant(o.status)}>{o.status}</Badge>
				</h2>
			</div>
			<div class="flex items-center gap-2">
				{#if allowed.includes('fulfilled')}
					<Button onclick={() => setStatus('fulfilled', 'fulfill')} loading={acting === 'fulfill'}>
						Mark fulfilled
					</Button>
				{/if}
				{#if allowed.includes('cancelled')}
					<Button variant="secondary" onclick={() => setStatus('cancelled', 'cancel')} loading={acting === 'cancel'}>
						Cancel
					</Button>
				{/if}
				{#if allowed.includes('refunded') && o.payment_id}
					<Button variant="danger" onclick={refund} loading={acting === 'refund'}>
						Refund
					</Button>
				{/if}
			</div>
		</div>

		<div class="grid grid-cols-3 gap-4">
			<Card>
				<h3 class="text-xs font-medium text-text-secondary mb-2">Customer</h3>
				<div class="space-y-1 text-sm">
					<div>{o.customer_name || '(no name)'}</div>
					<div class="text-text-muted">{o.customer_email}</div>
					{#if o.customer_phone}
						<div class="text-text-muted">{o.customer_phone}</div>
					{/if}
				</div>
			</Card>
			<Card>
				<h3 class="text-xs font-medium text-text-secondary mb-2">Payment</h3>
				<div class="space-y-1 text-xs">
					<div>Provider: <span class="font-mono">{o.payment_provider}</span></div>
					{#if o.payment_id}
						<div>Payment ID: <span class="font-mono">{o.payment_id}</span></div>
					{/if}
					{#if o.payment_status}
						<div>Mollie status: <span class="font-mono">{o.payment_status}</span></div>
					{/if}
					{#if o.refund_id}
						<div>Refund ID: <span class="font-mono">{o.refund_id}</span></div>
					{/if}
				</div>
			</Card>
			<Card>
				<h3 class="text-xs font-medium text-text-secondary mb-2">Totals</h3>
				<dl class="space-y-1 text-xs">
					<div class="flex justify-between">
						<dt class="text-text-muted">Subtotal</dt>
						<dd class="tabular-nums">{formatPrice(o.subtotal_cents, o.currency)}</dd>
					</div>
					{#if o.discount_cents > 0}
						<div class="flex justify-between text-text-muted">
							<dt>Discount</dt>
							<dd class="tabular-nums">-{formatPrice(o.discount_cents, o.currency)}</dd>
						</div>
					{/if}
					{#if o.tax_cents > 0}
						<div class="flex justify-between text-text-muted">
							<dt>Tax</dt>
							<dd class="tabular-nums">{formatPrice(o.tax_cents, o.currency)}</dd>
						</div>
					{/if}
					{#if o.shipping_cents > 0}
						<div class="flex justify-between text-text-muted">
							<dt>Shipping</dt>
							<dd class="tabular-nums">{formatPrice(o.shipping_cents, o.currency)}</dd>
						</div>
					{/if}
					<div class="flex justify-between border-t border-border-light pt-1 mt-1 text-sm font-medium">
						<dt>Total</dt>
						<dd class="tabular-nums">{formatPrice(o.total_cents, o.currency)}</dd>
					</div>
				</dl>
			</Card>
		</div>

		<Card>
			<h3 class="text-sm font-medium text-text-primary mb-3">Line items</h3>
			<div class="overflow-x-auto rounded-lg border border-border-light">
				<table class="w-full text-xs">
					<thead class="bg-bg-elevated text-text-muted">
						<tr>
							<th class="text-left font-normal px-3 py-2">Product</th>
							<th class="text-left font-normal px-3 py-2">Variant</th>
							<th class="text-left font-normal px-3 py-2">SKU</th>
							<th class="text-right font-normal px-3 py-2 tabular-nums">Qty</th>
							<th class="text-right font-normal px-3 py-2 tabular-nums">Unit</th>
							<th class="text-right font-normal px-3 py-2 tabular-nums">Total</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border-light">
						{#each data.items as it (it.id)}
							<tr>
								<td class="px-3 py-1.5">{it.product_name}</td>
								<td class="px-3 py-1.5 text-text-muted">{it.variant_name}</td>
								<td class="px-3 py-1.5 font-mono text-text-muted">{it.sku || '-'}</td>
								<td class="px-3 py-1.5 text-right tabular-nums">{it.quantity}</td>
								<td class="px-3 py-1.5 text-right tabular-nums">{formatPrice(it.unit_price_cents, o.currency)}</td>
								<td class="px-3 py-1.5 text-right tabular-nums">{formatPrice(it.total_cents, o.currency)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</Card>

		<Card>
			<h3 class="text-sm font-medium text-text-primary mb-3">Notes</h3>
			<textarea
				class="w-full h-24 rounded-lg border border-border bg-bg-elevated px-3 py-2 text-[13px] text-text-primary focus:outline-none focus:border-accent focus:ring-1 focus:ring-accent"
				placeholder="Internal notes (admin-only)"
				bind:value={notes}
			></textarea>
			<div class="mt-2 flex justify-end">
				<Button onclick={saveNotes} loading={savingNotes} disabled={notes === o.notes}>Save notes</Button>
			</div>
		</Card>
	</div>
{/if}

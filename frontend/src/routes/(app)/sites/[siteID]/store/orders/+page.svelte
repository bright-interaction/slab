<script lang="ts">
	import { onMount } from 'svelte';
	import { page as pageState } from '$app/state';
	import * as ordersApi from '$lib/api/orders';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import { ApiError } from '$lib/api/client';
	import type { Order, OrderStatus } from '$lib/api/orders';

	const siteID = $derived(pageState.params.siteID as string);

	let orders = $state<Order[]>([]);
	let loading = $state(true);
	let error = $state('');
	let filter = $state<OrderStatus | 'all'>('all');

	const filterTabs: { id: OrderStatus | 'all'; label: string }[] = [
		{ id: 'all', label: 'All' },
		{ id: 'pending', label: 'Pending' },
		{ id: 'paid', label: 'Paid' },
		{ id: 'fulfilled', label: 'Fulfilled' },
		{ id: 'refunded', label: 'Refunded' },
		{ id: 'failed', label: 'Failed' }
	];

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		error = '';
		try {
			orders = await ordersApi.listOrders(siteID, filter);
		} catch (err) {
			error = err instanceof ApiError ? err.message : 'Failed to load orders';
			orders = [];
		} finally {
			loading = false;
		}
	}

	async function setFilter(next: OrderStatus | 'all') {
		filter = next;
		await load();
	}

	function statusVariant(s: OrderStatus): 'success' | 'warning' | 'danger' | 'default' {
		switch (s) {
			case 'paid':
				return 'success';
			case 'fulfilled':
				return 'success';
			case 'pending':
				return 'warning';
			case 'cancelled':
			case 'failed':
				return 'danger';
			default:
				return 'default';
		}
	}

	function formatPrice(cents: number, currency: string): string {
		const sym = currency === 'EUR' ? '€' : currency === 'GBP' ? '£' : currency === 'USD' ? '$' : currency + ' ';
		return sym + (cents / 100).toFixed(2);
	}
</script>

<div class="space-y-4">
	<div class="flex items-center justify-between">
		<div>
			<h2 class="text-sm font-medium text-text-primary">Orders</h2>
			<p class="text-xs text-text-muted mt-1">
				{orders.length} order{orders.length === 1 ? '' : 's'} in the {filter === 'all' ? 'full history' : `"${filter}" state`}.
			</p>
		</div>
		<div class="flex items-center gap-1 text-xs">
			{#each filterTabs as t (t.id)}
				<button
					type="button"
					class="rounded px-2 py-1 text-text-muted hover:text-text-primary {filter === t.id ? 'font-medium text-text-primary' : ''}"
					onclick={() => setFilter(t.id)}
				>{t.label}</button>
			{/each}
		</div>
	</div>

	<Card>
		{#if loading}
			<div class="space-y-2"><Skeleton class="h-10" /><Skeleton class="h-10" /><Skeleton class="h-10" /></div>
		{:else if error}
			<div class="rounded-lg border border-danger/30 bg-danger/5 p-3 text-xs text-danger">{error}</div>
		{:else if orders.length === 0}
			<EmptyState
				title="No orders yet"
				description="Once a visitor completes a checkout, the order shows up here. To enable checkout, set payments.mollie_api_key in Settings with your tenant Mollie API key (test_xxx for staging, live_xxx for production)."
			/>
		{:else}
			<div class="overflow-x-auto rounded-lg border border-border-light">
				<table class="w-full text-xs">
					<thead class="bg-bg-elevated text-text-muted">
						<tr>
							<th class="text-left font-normal px-3 py-2">Order #</th>
							<th class="text-left font-normal px-3 py-2">Date</th>
							<th class="text-left font-normal px-3 py-2">Customer</th>
							<th class="text-left font-normal px-3 py-2">Status</th>
							<th class="text-right font-normal px-3 py-2 tabular-nums">Total</th>
							<th class="text-right font-normal px-3 py-2"></th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border-light">
						{#each orders as o (o.id)}
							<tr class="hover:bg-bg-hover transition-colors">
								<td class="px-3 py-1.5 font-mono">{o.order_number}</td>
								<td class="px-3 py-1.5 text-text-muted">{o.created_at}</td>
								<td class="px-3 py-1.5 truncate max-w-[32ch]">
									{o.customer_name || o.customer_email || '(anonymous)'}
								</td>
								<td class="px-3 py-1.5">
									<Badge variant={statusVariant(o.status)}>{o.status}</Badge>
								</td>
								<td class="px-3 py-1.5 text-right tabular-nums">{formatPrice(o.total_cents, o.currency)}</td>
								<td class="px-3 py-1.5 text-right">
									<a
										href={`/sites/${siteID}/store/orders/${o.id}`}
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

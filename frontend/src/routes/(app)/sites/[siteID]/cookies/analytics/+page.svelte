<script lang="ts">
	import * as consentApi from '$lib/api/consent';
	import { ApiError } from '$lib/api/client';
	import Card from '$lib/components/ui/Card.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import type { Site } from '$lib/api/types';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);

	let loading = $state(true);
	let stats = $state<consentApi.ConsentStats | null>(null);

	async function load() {
		loading = true;
		try {
			stats = await consentApi.stats(siteID);
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Failed to load consent stats.');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	function pct(part: number, total: number): string {
		if (total === 0) return '0%';
		return Math.round((part / total) * 100) + '%';
	}

	const dailyMax = $derived.by(() => {
		if (!stats || stats.daily.length === 0) return 1;
		let max = 0;
		for (const d of stats.daily) {
			if (d.total > max) max = d.total;
		}
		return Math.max(1, max);
	});

	function methodBarColor(method: string): string {
		switch (method) {
			case 'accept-all':
				return '#10b981';
			case 'reject-all':
				return '#f43f5e';
			case 'gpc':
				return '#8b5cf6';
			case 'custom':
				return '#f59e0b';
			default:
				return '#94a3b8';
		}
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
			Consent analytics
		</h1>
		<p class="text-[13px] text-text-secondary">
			Last 30 days. Aggregate splits by decision method and a daily time-series.
		</p>
	</header>

	{#if loading || !stats}
		<div class="mt-8 flex items-center justify-center py-12">
			<Spinner />
		</div>
	{:else}
		<div class="mt-6 grid grid-cols-2 gap-4 lg:grid-cols-4">
			<Card padding="md">
				<div class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Total proofs</div>
				<div class="mt-1 font-display text-3xl font-light text-text-primary">{stats.total}</div>
			</Card>
			<Card padding="md">
				<div class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Accept rate</div>
				<div class="mt-1 font-display text-3xl font-light text-emerald-600">
					{pct(stats.accepts, stats.total)}
				</div>
				<div class="mt-1 text-[11px] text-text-muted">{stats.accepts} accept-all</div>
			</Card>
			<Card padding="md">
				<div class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Reject rate</div>
				<div class="mt-1 font-display text-3xl font-light text-rose-600">
					{pct(stats.rejects, stats.total)}
				</div>
				<div class="mt-1 text-[11px] text-text-muted">{stats.rejects} reject-all</div>
			</Card>
			<Card padding="md">
				<div class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">GPC share</div>
				<div class="mt-1 font-display text-3xl font-light text-violet-600">
					{pct(stats.gpcs, stats.total)}
				</div>
				<div class="mt-1 text-[11px] text-text-muted">{stats.gpcs} signals honored</div>
			</Card>
		</div>

		<Card padding="md" class="mt-5">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Daily proofs (stacked by method)
			</h2>
			{#if stats.daily.length === 0}
				<p class="mt-4 text-[12px] text-text-muted">No proofs in the selected range.</p>
			{:else}
				<div class="mt-4 flex items-end gap-1.5 overflow-x-auto pb-2">
					{#each stats.daily as d (d.day)}
						<div class="flex min-w-[36px] flex-col items-center gap-1">
							<div class="flex h-32 w-7 flex-col-reverse rounded-t bg-bg-surface" title={`${d.day}: ${d.total}`}>
								{#each Object.entries(d.by_method) as [method, count]}
									<div
										class="w-full"
										style:height="{(count / dailyMax) * 100}%"
										style:background-color={methodBarColor(method)}
										title={`${method}: ${count}`}
									></div>
								{/each}
							</div>
							<div class="font-mono text-[10px] text-text-muted">{d.day.slice(5)}</div>
						</div>
					{/each}
				</div>
				<div class="mt-3 flex flex-wrap gap-3 text-[11px] text-text-muted">
					{#each ['accept-all', 'reject-all', 'custom', 'gpc'] as m}
						<span class="inline-flex items-center gap-1.5">
							<span class="h-2 w-3 rounded" style:background-color={methodBarColor(m)}></span>
							{m}
						</span>
					{/each}
				</div>
			{/if}
		</Card>

		<Card padding="md" class="mt-5">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Method distribution
			</h2>
			<div class="mt-4 flex flex-col gap-2">
				{#each [
					{ label: 'Accept all', count: stats.accepts, method: 'accept-all' },
					{ label: 'Reject all', count: stats.rejects, method: 'reject-all' },
					{ label: 'Custom', count: stats.customs, method: 'custom' },
					{ label: 'GPC', count: stats.gpcs, method: 'gpc' },
					{ label: 'DNS', count: stats.dns, method: 'dns' },
					{ label: 'Do Not Sell', count: stats.do_not_sell, method: 'do-not-sell' }
				] as r (r.method)}
					<div class="flex items-center gap-3">
						<span class="w-24 text-[12px] text-text-secondary">{r.label}</span>
						<div class="flex-1 h-3 rounded bg-bg-surface">
							<div
								class="h-full rounded"
								style:width={pct(r.count, stats.total)}
								style:background-color={methodBarColor(r.method)}
							></div>
						</div>
						<span class="w-12 text-right font-mono text-[11px] text-text-muted">{r.count}</span>
					</div>
				{/each}
			</div>
		</Card>
	{/if}
</div>

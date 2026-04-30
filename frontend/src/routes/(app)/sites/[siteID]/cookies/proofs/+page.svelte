<script lang="ts">
	import * as consentApi from '$lib/api/consent';
	import { ApiError } from '$lib/api/client';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import type { Site } from '$lib/api/types';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);

	const PAGE_SIZE = 50;

	let loading = $state(true);
	let records = $state<consentApi.ConsentRecord[]>([]);
	let total = $state(0);
	let offset = $state(0);

	let methodFilter = $state<string>('');
	let from = $state<string>('');
	let to = $state<string>('');

	function toMs(s: string): number | undefined {
		if (!s) return undefined;
		const d = new Date(s);
		if (isNaN(d.getTime())) return undefined;
		return d.getTime();
	}

	async function load() {
		loading = true;
		try {
			const res = await consentApi.list(siteID, {
				limit: PAGE_SIZE,
				offset,
				method: methodFilter || undefined,
				from: toMs(from),
				to: toMs(to)
			});
			records = res.records;
			total = res.total;
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Failed to load consent records.');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	function applyFilters() {
		offset = 0;
		void load();
	}

	function nextPage() {
		if (offset + PAGE_SIZE < total) {
			offset += PAGE_SIZE;
			void load();
		}
	}

	function prevPage() {
		if (offset > 0) {
			offset = Math.max(0, offset - PAGE_SIZE);
			void load();
		}
	}

	function exportUrl(): string {
		return consentApi.exportCsvUrl(siteID, { from: toMs(from), to: toMs(to) });
	}

	function fmtTime(s: string): string {
		try {
			return new Date(s).toLocaleString();
		} catch {
			return s;
		}
	}

	function categoriesLabel(c: Record<string, boolean | undefined>): string {
		const accepted = Object.entries(c)
			.filter(([, v]) => v === true)
			.map(([k]) => k);
		if (accepted.length === 0) return 'none';
		return accepted.join(', ');
	}

	function methodBadgeClass(method: string): string {
		switch (method) {
			case 'accept-all':
				return 'bg-emerald-100 text-emerald-800';
			case 'reject-all':
				return 'bg-rose-100 text-rose-800';
			case 'gpc':
				return 'bg-violet-100 text-violet-800';
			case 'custom':
				return 'bg-amber-100 text-amber-900';
			default:
				return 'bg-bg-surface text-text-muted';
		}
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
			Consent proofs
		</h1>
		<p class="text-[13px] text-text-secondary">
			GDPR audit log. Every consent decision visitors made on this site, with hashed IP and the
			page URL. Exportable as CSV for compliance reviews.
		</p>
	</header>

	<Card padding="md" class="mt-6">
		<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
			<label class="flex flex-col gap-1 text-[12px] text-text-muted">
				From
				<input
					type="date"
					class="rounded border border-border-light bg-bg px-2.5 py-1.5 text-[13px] text-text-primary"
					bind:value={from}
				/>
			</label>
			<label class="flex flex-col gap-1 text-[12px] text-text-muted">
				To
				<input
					type="date"
					class="rounded border border-border-light bg-bg px-2.5 py-1.5 text-[13px] text-text-primary"
					bind:value={to}
				/>
			</label>
			<label class="flex flex-col gap-1 text-[12px] text-text-muted">
				Method
				<select
					class="rounded border border-border-light bg-bg px-2.5 py-1.5 text-[13px] text-text-primary"
					bind:value={methodFilter}
				>
					<option value="">All</option>
					<option value="accept-all">Accept all</option>
					<option value="reject-all">Reject all</option>
					<option value="custom">Custom</option>
					<option value="gpc">GPC</option>
					<option value="dns">DNS</option>
					<option value="do-not-sell">Do Not Sell</option>
				</select>
			</label>
			<div class="flex items-end gap-2">
				<Button variant="primary" onclick={applyFilters}>Apply</Button>
				<a
					href={exportUrl()}
					class="inline-flex items-center justify-center rounded border border-border-light px-3 py-1.5 text-[13px] text-text-primary hover:bg-bg-surface"
				>
					Export CSV
				</a>
			</div>
		</div>
	</Card>

	{#if loading}
		<div class="mt-8 flex items-center justify-center py-12">
			<Spinner />
		</div>
	{:else}
		<Card padding="none" class="mt-6">
			<div class="overflow-x-auto">
				<table class="w-full text-[12px]">
					<thead class="border-b border-border-light bg-bg-surface/40 text-text-muted">
						<tr>
							<th class="px-4 py-2 text-left font-mono text-[11px] uppercase tracking-wider">Time</th>
							<th class="px-4 py-2 text-left font-mono text-[11px] uppercase tracking-wider">Method</th>
							<th class="px-4 py-2 text-left font-mono text-[11px] uppercase tracking-wider">Categories</th>
							<th class="px-4 py-2 text-left font-mono text-[11px] uppercase tracking-wider">Page</th>
							<th class="px-4 py-2 text-left font-mono text-[11px] uppercase tracking-wider">GPC</th>
							<th class="px-4 py-2 text-left font-mono text-[11px] uppercase tracking-wider">IP hash</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border-light">
						{#each records as r (r.id)}
							<tr class="hover:bg-bg-surface/40">
								<td class="px-4 py-2 text-text-primary">{fmtTime(r.created_at)}</td>
								<td class="px-4 py-2">
									<span class="inline-flex rounded px-2 py-0.5 text-[11px] font-medium {methodBadgeClass(r.method)}">{r.method}</span>
								</td>
								<td class="px-4 py-2 text-text-secondary">{categoriesLabel(r.categories)}</td>
								<td class="px-4 py-2 font-mono text-[11px] text-text-muted">{r.page}</td>
								<td class="px-4 py-2">{r.gpc_active ? '✓' : ''}</td>
								<td class="px-4 py-2 font-mono text-[11px] text-text-muted">{r.ip_hash_trunc}</td>
							</tr>
						{:else}
							<tr>
								<td colspan="6" class="px-4 py-8 text-center text-text-muted">No consent records in this range.</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
			<div class="flex items-center justify-between border-t border-border-light px-4 py-3 text-[12px] text-text-muted">
				<span>{total} total records</span>
				<div class="flex items-center gap-2">
					<Button variant="ghost" onclick={prevPage} disabled={offset === 0}>Previous</Button>
					<span>Showing {offset + 1}–{Math.min(offset + records.length, total)}</span>
					<Button variant="ghost" onclick={nextPage} disabled={offset + PAGE_SIZE >= total}>Next</Button>
				</div>
			</div>
		</Card>
	{/if}
</div>

<script lang="ts">
	import { ArrowLeft, Save, AlertTriangle, ShieldAlert } from 'lucide-svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Switch from '$lib/components/ui/Switch.svelte';
	import { ApiError } from '$lib/api/client';
	import * as quotaApi from '$lib/api/quota';
	import type { PerSiteQuota } from '$lib/api/quota';
	import { auth } from '$lib/stores/auth.svelte';
	import { toast } from '$lib/stores/toast.svelte';

	const isAdmin = $derived(auth.value?.role === 'admin');

	let rows = $state<PerSiteQuota[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let savingID = $state<string | null>(null);

	type EditState = {
		storage_quota_bytes: number;
		build_minutes_quota: number;
		quota_overage_blocked: boolean;
	};

	let edits = $state<Record<string, EditState>>({});

	async function load(): Promise<void> {
		loading = true;
		loadError = null;
		try {
			const res = await quotaApi.adminMetrics();
			rows = res.per_site_quotas ?? [];
			for (const row of rows) {
				edits[row.site_id] = {
					storage_quota_bytes: row.storage_quota_bytes,
					build_minutes_quota: row.build_minutes_quota,
					quota_overage_blocked: row.blocked_on_overage
				};
			}
		} catch (err) {
			loadError = err instanceof ApiError ? err.message : 'Failed to load metrics.';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		if (isAdmin) void load();
	});

	function fmtBytes(n: number): string {
		if (n >= 1024 * 1024 * 1024) return (n / 1024 / 1024 / 1024).toFixed(2) + ' GiB';
		if (n >= 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + ' MiB';
		if (n >= 1024) return (n / 1024).toFixed(0) + ' KiB';
		return n + ' B';
	}

	function fmtPct(p: number): string {
		return (p * 100).toFixed(1) + '%';
	}

	function pctBarClass(p: number): string {
		if (p >= 0.9) return 'bg-danger';
		if (p >= 0.75) return 'bg-amber-500';
		return 'bg-accent';
	}

	async function save(siteID: string): Promise<void> {
		const e = edits[siteID];
		if (!e || savingID) return;
		savingID = siteID;
		try {
			await quotaApi.patchAdminQuota(siteID, {
				storage_quota_bytes: e.storage_quota_bytes,
				build_minutes_quota: e.build_minutes_quota,
				quota_overage_blocked: e.quota_overage_blocked
			});
			toast.success('Quota updated.');
			await load();
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Failed to save.');
		} finally {
			savingID = null;
		}
	}
</script>

{#if !isAdmin}
	<div class="mx-auto max-w-5xl px-6 py-8">
		<Card padding="md">
			<div class="flex items-start gap-3">
				<ShieldAlert size={20} strokeWidth={1.5} class="text-danger" />
				<div>
					<h1 class="text-[15px] font-medium text-text-primary">Admin only</h1>
					<p class="mt-1 text-[13px] text-text-secondary">This page is restricted to workspace admins.</p>
				</div>
			</div>
		</Card>
	</div>
{:else}
	<div class="mx-auto max-w-7xl px-6 py-8">
		<a
			href="/"
			class="inline-flex items-center gap-1.5 text-[12px] text-text-muted transition-colors hover:text-text-primary"
		>
			<ArrowLeft size={14} strokeWidth={1.75} />
			Dashboard
		</a>
		<header class="mt-3 flex flex-col gap-1.5">
			<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">Per-tenant quotas</h1>
			<p class="text-[13px] text-text-secondary">
				Storage caps total media bytes per site. Build minutes cap the rolling 30-day sum of completed deployment durations.
			</p>
		</header>

		{#if loading}
			<p class="mt-8 text-[12px] text-text-muted">Loading...</p>
		{:else if loadError}
			<Card padding="md" class="mt-8 border-danger/30">
				<div class="flex items-start gap-3">
					<AlertTriangle size={16} strokeWidth={1.5} class="mt-0.5 text-danger" />
					<p class="text-[13px] text-danger">{loadError}</p>
				</div>
			</Card>
		{:else if rows.length === 0}
			<p class="mt-8 text-[12px] text-text-muted">No sites yet.</p>
		{:else}
			<div class="mt-8 overflow-hidden rounded-2xl border border-border-light bg-bg-surface">
				<table class="w-full text-[13px]">
					<thead class="bg-bg-elevated text-left text-[11px] font-medium uppercase tracking-[0.18em] text-text-muted">
						<tr>
							<th class="px-4 py-3">Site</th>
							<th class="px-4 py-3">Storage usage</th>
							<th class="px-4 py-3">Storage quota (bytes)</th>
							<th class="px-4 py-3">Build minutes (30d)</th>
							<th class="px-4 py-3">Build minutes quota</th>
							<th class="px-4 py-3">Block on overage</th>
							<th class="px-4 py-3"></th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border-light">
						{#each rows as row (row.site_id)}
							<tr>
								<td class="px-4 py-3">
									<div class="font-medium text-text-primary">{row.name}</div>
									<div class="font-mono text-[11px] text-text-muted">/{row.slug}</div>
								</td>
								<td class="px-4 py-3">
									<div class="text-text-primary">{fmtBytes(row.storage_used_bytes)}</div>
									<div class="mt-1 h-1.5 w-32 overflow-hidden rounded-full bg-bg-elevated">
										<div class="h-full {pctBarClass(row.storage_pct)}" style="width: {Math.min(100, row.storage_pct * 100)}%"></div>
									</div>
									<div class="mt-0.5 font-mono text-[10px] text-text-muted">{fmtPct(row.storage_pct)}</div>
								</td>
								<td class="px-4 py-3">
									<Input type="number" bind:value={edits[row.site_id].storage_quota_bytes} hint={fmtBytes(edits[row.site_id]?.storage_quota_bytes ?? 0)} />
								</td>
								<td class="px-4 py-3">
									<div class="text-text-primary">{row.build_minutes_used}</div>
									<div class="mt-1 h-1.5 w-32 overflow-hidden rounded-full bg-bg-elevated">
										<div class="h-full {pctBarClass(row.build_minutes_pct)}" style="width: {Math.min(100, row.build_minutes_pct * 100)}%"></div>
									</div>
									<div class="mt-0.5 font-mono text-[10px] text-text-muted">{fmtPct(row.build_minutes_pct)}</div>
								</td>
								<td class="px-4 py-3">
									<Input type="number" bind:value={edits[row.site_id].build_minutes_quota} />
								</td>
								<td class="px-4 py-3">
									<Switch bind:checked={edits[row.site_id].quota_overage_blocked} />
								</td>
								<td class="px-4 py-3">
									<Button variant="secondary" size="sm" loading={savingID === row.site_id} disabled={savingID === row.site_id} onclick={() => save(row.site_id)}>
										<Save size={12} strokeWidth={1.75} />Save
									</Button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>
{/if}

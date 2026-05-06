<script lang="ts">
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Switch from '$lib/components/ui/Switch.svelte';
	import Dialog from '$lib/components/ui/Dialog.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import * as migrations from '$lib/api/migrations';
	import { ApiError } from '$lib/api/client';
	import type { Migration, URLPlan, ApplyResult, PlannerOptions } from '$lib/api/migrations';

	let {
		siteID,
		migration,
		plan,
		options,
		onApplied
	}: {
		siteID: string;
		migration: Migration;
		plan: URLPlan;
		options: PlannerOptions;
		onApplied: (result: ApplyResult) => void;
	} = $props();

	let upsert = $state(false);
	let confirmOpen = $state(false);
	let applying = $state(false);
	let quotaError = $state<{ message: string; current?: number; projected?: number; limit?: number } | null>(null);

	const conflictCount = $derived(plan.conflicts?.length ?? 0);
	// Apply is gated by conflicts UNLESS upsert is on (upsert tells the
	// backend to pass an empty existing-slugs map to PlanURLs, so the
	// recomputed server-side plan won't carry conflicts at all).
	const applyDisabled = $derived(
		applying || migration.status === 'applied' || (conflictCount > 0 && !upsert)
	);

	const newPageCount = $derived((plan.pages ?? []).filter((p) => !p.skipped).length);
	const newRedirectCount = $derived((plan.redirects ?? []).length);

	async function apply() {
		quotaError = null;
		applying = true;
		try {
			const result = await migrations.applyMigration(siteID, migration.id, {
				...options,
				upsert
			});
			const created =
				result.pages_created + result.redirects_created + result.items_created;
			const updated =
				(result.pages_updated ?? 0) +
				(result.redirects_updated ?? 0) +
				(result.items_updated ?? 0);
			toast.success(
				upsert
					? `Refresh applied: ${created} created, ${updated} updated`
					: `Migration applied: ${created} rows`
			);
			onApplied(result);
			confirmOpen = false;
		} catch (err) {
			if (err instanceof ApiError && err.status === 402) {
				quotaError = { message: err.message };
				toast.error('Plan cap reached, see details below');
			} else {
				toast.error(err instanceof ApiError ? err.message : 'Apply failed');
			}
		} finally {
			applying = false;
		}
	}
</script>

<Card>
	<div class="flex items-baseline justify-between mb-3">
		<h2 class="text-sm font-medium text-text-primary">Apply</h2>
		{#if migration.status === 'applied'}
			<span class="text-xs text-text-muted">Already applied. Toggle upsert to refresh.</span>
		{/if}
	</div>

	<div class="space-y-4">
		<div class="flex items-start justify-between gap-4">
			<div>
				<div class="flex items-center gap-3">
					<Switch bind:checked={upsert} label="Upsert mode" ariaLabel="Upsert mode" />
					<span class="text-[13px] text-text-primary">Re-import / refresh mode</span>
				</div>
				<p class="text-xs text-text-muted mt-1.5 max-w-prose">
					Off (default): create-only. The porter errors on slug conflicts.
					On: existing pages, redirects, and items get UPDATED in place by their natural key.
					Use for annual content refreshes against the same site.
				</p>
			</div>
		</div>

		{#if quotaError}
			<div class="rounded-lg border border-danger/30 bg-danger/5 px-3 py-2 text-xs text-danger">
				<strong>Plan cap reached.</strong> {quotaError.message}
				<div class="mt-1 text-text-muted">
					The migration row stays at status="ready" so you can adjust skip paths or upgrade the
					workspace plan and retry.
				</div>
			</div>
		{/if}

		<div class="flex items-center justify-between border-t border-border-light pt-4">
			<div class="text-xs text-text-muted">
				Will commit
				<strong class="text-text-primary">{newPageCount}</strong> pages +
				<strong class="text-text-primary">{newRedirectCount}</strong> redirects
				{#if conflictCount > 0 && !upsert}
					<span class="text-danger">({conflictCount} conflict{conflictCount === 1 ? '' : 's'} blocks Apply)</span>
				{/if}
			</div>
			<Button onclick={() => (confirmOpen = true)} disabled={applyDisabled}>
				{upsert ? 'Apply (refresh)' : 'Apply migration'}
			</Button>
		</div>
	</div>
</Card>

<Dialog
	bind:open={confirmOpen}
	title={upsert ? 'Apply with upsert mode?' : 'Apply migration?'}
	description={upsert
		? `This will UPDATE existing pages, redirects, and items by natural key. Existing content with overlapping slugs will be overwritten. ${newPageCount} candidate pages.`
		: `This will create ${newPageCount} pages and ${newRedirectCount} redirects on the live site. The migration row will flip to "applied".`}
>
	{#snippet footer()}
		<div class="flex justify-end gap-2">
			<Button variant="ghost" onclick={() => (confirmOpen = false)} disabled={applying}>
				Cancel
			</Button>
			<Button onclick={apply} loading={applying}>
				{upsert ? 'Apply refresh' : 'Apply migration'}
			</Button>
		</div>
	{/snippet}
</Dialog>

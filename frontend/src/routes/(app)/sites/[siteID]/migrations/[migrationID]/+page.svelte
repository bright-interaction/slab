<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Tabs from '$lib/components/ui/Tabs.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';
	import MigrationManifestSummary from '$lib/components/migrations/MigrationManifestSummary.svelte';
	import URLPlanTable from '$lib/components/migrations/URLPlanTable.svelte';
	import ApplyButton from '$lib/components/migrations/ApplyButton.svelte';
	import VerifyLivePanel from '$lib/components/migrations/VerifyLivePanel.svelte';
	import MissingURLsPanel from '$lib/components/migrations/MissingURLsPanel.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import * as migrations from '$lib/api/migrations';
	import { ApiError } from '$lib/api/client';
	import type { Site } from '$lib/api/types';
	import type { Migration, URLPlan, PlannerOptions, ApplyResult } from '$lib/api/migrations';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);
	const migrationID = $derived(page.params.migrationID ?? '');

	let migration = $state<Migration | null>(null);
	let plan = $state<URLPlan | null>(null);
	let loading = $state(true);
	let recomputing = $state(false);
	let plannerOptions = $state<PlannerOptions>({
		skip_paths: [],
		collapse_date_archives: true,
		preserve_locale_prefix: false
	});

	let activeTab = $state('verify');
	let deleteOpen = $state(false);
	let deleting = $state(false);

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		try {
			migration = await migrations.getMigration(siteID, migrationID);
			await recomputePlan();
		} catch (err) {
			if (err instanceof ApiError && err.status === 404) {
				toast.error('Migration not found');
				void goto(`/sites/${siteID}/migrations`);
				return;
			}
			toast.error(err instanceof ApiError ? err.message : 'Load failed');
		} finally {
			loading = false;
		}
	}

	async function recomputePlan() {
		recomputing = true;
		try {
			plan = await migrations.planMigration(siteID, migrationID, plannerOptions);
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Plan failed');
		} finally {
			recomputing = false;
		}
	}

	async function onApplied(_: ApplyResult) {
		// Reload migration so the status badge updates from "ready" to "applied".
		try {
			migration = await migrations.getMigration(siteID, migrationID);
		} catch {
			// Already toasted by ApplyButton; the page state stays usable.
		}
	}

	async function deleteMigration() {
		deleting = true;
		try {
			await migrations.deleteMigration(siteID, migrationID);
			toast.success('Migration deleted');
			void goto(`/sites/${siteID}/migrations`);
		} catch (err) {
			toast.error(err instanceof ApiError ? err.message : 'Delete failed');
		} finally {
			deleting = false;
		}
	}

	function statusVariant(s: string): 'default' | 'success' | 'warning' | 'danger' | 'info' {
		switch (s) {
			case 'applied':
				return 'success';
			case 'failed':
				return 'danger';
			case 'ready':
				return 'info';
			case 'pending':
				return 'warning';
			default:
				return 'default';
		}
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<div class="mb-6 flex items-center justify-between">
		<div>
			<a
				href="/sites/{siteID}/migrations"
				class="text-xs text-text-muted hover:text-text-primary inline-flex items-center gap-1"
			>← All migrations</a>
			<h1 class="text-2xl font-light tracking-tight text-text-primary mt-1 truncate max-w-3xl">
				{migration?.source_url ?? 'Migration'}
			</h1>
		</div>
		<div class="flex items-center gap-2">
			{#if migration}
				<Badge variant={statusVariant(migration.status)}>{migration.status}</Badge>
				<span class="text-xs text-text-muted">{migration.source_type}</span>
				<Button variant="danger" size="sm" onclick={() => (deleteOpen = true)}>Delete</Button>
			{/if}
		</div>
	</div>

	{#if loading}
		<div class="space-y-4">
			<Skeleton class="h-32" />
			<Skeleton class="h-64" />
			<Skeleton class="h-32" />
		</div>
	{:else if migration && plan}
		<div class="space-y-6">
			{#if migration.error}
				<Card>
					<div class="flex items-start gap-3">
						<Badge variant="danger">error</Badge>
						<p class="text-xs text-danger flex-1">{migration.error}</p>
					</div>
				</Card>
			{/if}

			<MigrationManifestSummary {migration} />

			<URLPlanTable
				{plan}
				bind:options={plannerOptions}
				{recomputing}
				onRecompute={recomputePlan}
			/>

			<ApplyButton {siteID} {migration} {plan} options={plannerOptions} {onApplied} />

			<Tabs
				tabs={[
					{ id: 'verify', label: 'Verify live' },
					{ id: 'missing', label: '404 inbox' }
				]}
				bind:value={activeTab}
			/>

			{#if activeTab === 'verify'}
				<VerifyLivePanel {siteID} migrationID={migration.id} />
			{:else if activeTab === 'missing'}
				<MissingURLsPanel {siteID} />
			{/if}
		</div>
	{/if}
</div>

<ConfirmDialog
	bind:open={deleteOpen}
	title="Delete this migration?"
	message="The migration row will be removed. Pages, redirects, and items already imported from a previously-applied migration are NOT touched. Safe to delete; you can re-import via a fresh crawl."
	confirmText="Delete"
	variant="danger"
	onconfirm={deleteMigration}
/>

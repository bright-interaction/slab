<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import NewMigrationForm from '$lib/components/migrations/NewMigrationForm.svelte';
	import * as migrations from '$lib/api/migrations';
	import type { Site } from '$lib/api/types';
	import type { MigrationSummary } from '$lib/api/migrations';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);

	let rows = $state<MigrationSummary[]>([]);
	let loading = $state(true);
	let formOpen = $state(false);

	onMount(() => {
		void load();
	});

	async function load() {
		loading = true;
		try {
			rows = await migrations.listMigrations(siteID);
		} catch {
			rows = [];
		} finally {
			loading = false;
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

	function onCreated(migrationID: string) {
		formOpen = false;
		void goto(`/sites/${siteID}/migrations/${migrationID}`);
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<div class="mb-6">
		<h1 class="text-2xl font-light tracking-tight text-text-primary">Migrations</h1>
		<p class="mt-1 text-sm text-text-muted">
			Crawl a source CMS, review the URL plan, apply pages + redirects + media in one shot, then
			verify the live site after DNS flips.
		</p>
	</div>

	<div class="space-y-6">
		{#if formOpen}
			<NewMigrationForm {siteID} onSuccess={onCreated} />
			<div class="flex justify-end">
				<button
					type="button"
					class="text-xs text-text-muted hover:text-text-primary"
					onclick={() => (formOpen = false)}
				>Cancel</button>
			</div>
		{:else}
			<Card>
				<div class="flex items-center justify-between">
					<div>
						<h2 class="text-sm font-medium text-text-primary">Start a new migration</h2>
						<p class="text-xs text-text-muted mt-1">
							Sitemap, WordPress, Webflow, or Ghost.
						</p>
					</div>
					<button
						type="button"
						class="inline-flex items-center h-9 px-4 text-[13px] gap-2 rounded-lg bg-accent text-accent-fg hover:bg-accent-hover font-medium"
						onclick={() => (formOpen = true)}
					>+ New migration</button>
				</div>
			</Card>
		{/if}

		<Card>
			<h2 class="text-sm font-medium text-text-primary mb-4">Past migrations</h2>
			{#if loading}
				<div class="space-y-2">
					<Skeleton class="h-10" />
					<Skeleton class="h-10" />
					<Skeleton class="h-10" />
				</div>
			{:else if rows.length === 0}
				<EmptyState
					title="No migrations yet"
					description="Start your first crawl above to import an existing CMS."
				/>
			{:else}
				<div class="overflow-x-auto rounded-lg border border-border-light">
					<table class="w-full text-xs">
						<thead class="bg-bg-elevated text-text-muted">
							<tr>
								<th class="text-left font-normal px-3 py-2">Source</th>
								<th class="text-left font-normal px-3 py-2">Type</th>
								<th class="text-left font-normal px-3 py-2">Status</th>
								<th class="text-left font-normal px-3 py-2 tabular-nums">Pages</th>
								<th class="text-left font-normal px-3 py-2">Created</th>
								<th class="text-right font-normal px-3 py-2"></th>
							</tr>
						</thead>
						<tbody class="divide-y divide-border-light">
							{#each rows as row (row.id)}
								<tr class="hover:bg-bg-hover transition-colors">
									<td class="px-3 py-1.5 font-mono truncate max-w-[40ch]">{row.source_url}</td>
									<td class="px-3 py-1.5 text-text-muted">{row.source_type}</td>
									<td class="px-3 py-1.5">
										<Badge variant={statusVariant(row.status)}>{row.status}</Badge>
									</td>
									<td class="px-3 py-1.5 tabular-nums">{row.stats?.pages ?? 0}</td>
									<td class="px-3 py-1.5 text-text-muted">{row.created_at}</td>
									<td class="px-3 py-1.5 text-right">
										<a
											href="/sites/{siteID}/migrations/{row.id}"
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
</div>

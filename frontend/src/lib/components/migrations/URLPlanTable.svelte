<script lang="ts">
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Switch from '$lib/components/ui/Switch.svelte';
	import Textarea from '$lib/components/ui/Textarea.svelte';
	import type { URLPlan, PlannerOptions } from '$lib/api/migrations';

	let {
		plan,
		options = $bindable(),
		recomputing = false,
		onRecompute
	}: {
		plan: URLPlan;
		options: PlannerOptions;
		recomputing?: boolean;
		onRecompute: () => void;
	} = $props();

	let skipPathsRaw = $state((options.skip_paths ?? []).join('\n'));
	let collapseDateArchives = $state(options.collapse_date_archives ?? true);
	let preserveLocalePrefix = $state(options.preserve_locale_prefix ?? false);

	const livePages = $derived(plan.pages?.filter((p) => !p.skipped) ?? []);
	const skippedPages = $derived(plan.pages?.filter((p) => p.skipped) ?? []);
	const conflictCount = $derived(plan.conflicts?.length ?? 0);
	const redirectCount = $derived(plan.redirects?.length ?? 0);

	function recompute() {
		options.skip_paths = skipPathsRaw
			.split(/\r?\n|,/)
			.map((s) => s.trim())
			.filter(Boolean);
		options.collapse_date_archives = collapseDateArchives;
		options.preserve_locale_prefix = preserveLocalePrefix;
		onRecompute();
	}
</script>

<Card>
	<div class="flex items-baseline justify-between mb-4">
		<h2 class="text-sm font-medium text-text-primary">URL plan</h2>
		<div class="flex items-center gap-3 text-xs text-text-muted">
			<span>{livePages.length} live</span>
			{#if skippedPages.length > 0}
				<span>{skippedPages.length} skipped</span>
			{/if}
			<span>{redirectCount} redirects</span>
			{#if conflictCount > 0}
				<Badge variant="danger">{conflictCount} conflict{conflictCount === 1 ? '' : 's'}</Badge>
			{/if}
		</div>
	</div>

	<details class="mb-4 group">
		<summary class="cursor-pointer text-xs text-text-secondary hover:text-text-primary list-none flex items-center gap-1.5">
			<svg class="h-3 w-3 transition-transform group-open:rotate-90" fill="none" stroke="currentColor" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
			</svg>
			Planner options
		</summary>
		<div class="mt-3 space-y-3 pl-5">
			<div>
				<label for="skip_paths" class="block text-xs text-text-muted mb-1.5">
					Skip paths (one per line; manifest source paths to drop from the import)
				</label>
				<Textarea
					id="skip_paths"
					bind:value={skipPathsRaw}
					rows={3}
					placeholder={'/2020/01/old-post\n/dev/test-page'}
				/>
			</div>
			<div class="flex items-center gap-6">
				<Switch
					bind:checked={collapseDateArchives}
					label="Collapse date archives"
					ariaLabel="Collapse date archives"
				/>
				<span class="text-xs text-text-muted">/2024/01/post → /post</span>
			</div>
			<div class="flex items-center gap-6">
				<Switch
					bind:checked={preserveLocalePrefix}
					label="Preserve locale prefix"
					ariaLabel="Preserve locale prefix"
				/>
				<span class="text-xs text-text-muted">/sv/blog stays /sv/blog</span>
			</div>
			<Button size="sm" variant="secondary" onclick={recompute} loading={recomputing}>
				Recompute plan
			</Button>
		</div>
	</details>

	{#if conflictCount > 0}
		<div class="mb-4 rounded-lg border border-danger/30 bg-danger/5 px-3 py-2 text-xs text-danger">
			<strong>{conflictCount} conflict{conflictCount === 1 ? '' : 's'}</strong> with live pages.
			Adjust skip paths or use upsert mode (which overwrites by natural key) to apply anyway.
		</div>
	{/if}

	<div class="overflow-x-auto rounded-lg border border-border-light">
		<table class="w-full text-xs">
			<thead class="bg-bg-elevated text-text-muted">
				<tr>
					<th class="text-left font-normal px-3 py-2">Source path</th>
					<th class="text-left font-normal px-3 py-2">→</th>
					<th class="text-left font-normal px-3 py-2">New slug</th>
					<th class="text-left font-normal px-3 py-2">Title</th>
					<th class="text-left font-normal px-3 py-2"></th>
				</tr>
			</thead>
			<tbody class="divide-y divide-border-light">
				{#each plan.pages ?? [] as p (p.source_path)}
					<tr class={p.skipped ? 'text-text-muted' : 'text-text-primary'}>
						<td class="px-3 py-1.5 font-mono">{p.source_path}</td>
						<td class="px-3 py-1.5 text-text-muted">→</td>
						<td class="px-3 py-1.5 font-mono">{p.new_slug || '/'}</td>
						<td class="px-3 py-1.5 truncate max-w-[24ch]">{p.title}</td>
						<td class="px-3 py-1.5">
							{#if p.skipped}
								<Badge variant="default">skipped{p.skip_reason ? `: ${p.skip_reason}` : ''}</Badge>
							{/if}
						</td>
					</tr>
				{:else}
					<tr>
						<td colspan="5" class="px-3 py-6 text-center text-text-muted">No pages in plan</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>

	{#if redirectCount > 0}
		<details class="mt-4 group">
			<summary class="cursor-pointer text-xs text-text-secondary hover:text-text-primary list-none flex items-center gap-1.5">
				<svg class="h-3 w-3 transition-transform group-open:rotate-90" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
				{redirectCount} planned redirect{redirectCount === 1 ? '' : 's'}
			</summary>
			<div class="mt-3 overflow-x-auto rounded-lg border border-border-light">
				<table class="w-full text-xs">
					<thead class="bg-bg-elevated text-text-muted">
						<tr>
							<th class="text-left font-normal px-3 py-2">From</th>
							<th class="text-left font-normal px-3 py-2">→</th>
							<th class="text-left font-normal px-3 py-2">To</th>
							<th class="text-left font-normal px-3 py-2">Status</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border-light">
						{#each plan.redirects ?? [] as r (r.from_path)}
							<tr>
								<td class="px-3 py-1.5 font-mono">{r.from_path}</td>
								<td class="px-3 py-1.5 text-text-muted">→</td>
								<td class="px-3 py-1.5 font-mono">{r.to_path}</td>
								<td class="px-3 py-1.5 tabular-nums">{r.status_code}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</details>
	{/if}
</Card>

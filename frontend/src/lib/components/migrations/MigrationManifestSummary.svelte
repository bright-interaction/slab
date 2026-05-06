<script lang="ts">
	import Card from '$lib/components/ui/Card.svelte';
	import type { Migration } from '$lib/api/migrations';

	let { migration }: { migration: Migration } = $props();

	const stats = $derived(migration.manifest?.stats ?? {
		pages_found: 0,
		collections_found: 0,
		items_found: 0,
		media_found: 0
	});

	const cells = $derived([
		{ label: 'Pages', value: stats.pages_found ?? 0 },
		{ label: 'Collections', value: stats.collections_found ?? 0 },
		{ label: 'Items', value: stats.items_found ?? 0 },
		{ label: 'Media', value: stats.media_found ?? 0 }
	]);
</script>

<Card>
	<h2 class="text-sm font-medium text-text-primary mb-4">Manifest summary</h2>
	<div class="grid grid-cols-4 gap-4">
		{#each cells as cell (cell.label)}
			<div class="border-l border-border-light pl-4 first:border-l-0 first:pl-0">
				<div class="text-2xl font-light text-text-primary tabular-nums">{cell.value}</div>
				<div class="text-xs text-text-muted mt-1">{cell.label}</div>
			</div>
		{/each}
	</div>
</Card>

<script lang="ts">
	import ClarificationsPanel from '$lib/components/clarifications/ClarificationsPanel.svelte';
	import type { Site } from '$lib/api/types';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);
</script>

<div class="mx-auto max-w-7xl px-6 py-8 space-y-6">
	<div>
		<h1 class="text-2xl font-light tracking-tight text-text-primary">Clarifications</h1>
		<p class="mt-1 text-sm text-text-muted">
			Open questions from the build agent. Resolve so it can keep going.
		</p>
	</div>
	<ClarificationsPanel {siteID} />
</div>

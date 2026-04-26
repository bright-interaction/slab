<script lang="ts">
	import { goto } from '$app/navigation';
	import type { Site } from '$lib/api/types';

	// The Evals tab was merged into Build (Phase 12.7). Anyone hitting the
	// old URL gets redirected to the unified page. Direct deep-links to
	// /sites/{id}/evaluations/{buildID} continue to work via the sibling
	// route file (per-build detail page is unchanged URL-wise).
	let { data }: { data: { site: Site } } = $props();
	const siteID = $derived(data.site.id);

	$effect(() => {
		void goto(`/sites/${siteID}/build`, { replaceState: true });
	});
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<p class="text-[13px] text-text-muted">Redirecting to Build...</p>
</div>

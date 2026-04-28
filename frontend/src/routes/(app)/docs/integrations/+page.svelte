<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	// Phase 19: /docs/integrations folded into /docs/apps#platform.
	// Both surfaces were "stuff the platform connects to". Apps was the
	// outward-facing connector catalogue (CRMs, analytics, banners), while
	// Integrations described platform-level mechanics (DNS, fonts, eval
	// engine, CI). Reading two pages to cover one mental model created
	// drift; merging the platform section into Apps gives one canonical
	// reference. Old bookmarks land on the Platform anchor.
	onMount(() => {
		void goto('/docs/apps#platform', { replaceState: true });
	});
</script>

<div class="mx-auto max-w-5xl px-6 py-12 text-center">
	<p class="text-[13px] text-text-secondary">
		Integrations have moved into
		<a href="/docs/apps#platform" class="text-text-primary hover:underline">Apps & integrations</a>.
		Redirecting...
	</p>
</div>

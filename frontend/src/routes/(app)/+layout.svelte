<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
	import Topbar from '$lib/components/layout/Topbar.svelte';
	import CommandPalette from '$lib/components/ui/CommandPalette.svelte';
	import { currentSite } from '$lib/stores/currentSite.svelte';

	let { children }: { children: Snippet } = $props();

	const SITE_ROUTE = /^\/sites\/([^/]+)(?:\/.*)?$/;

	const siteIdFromRoute = $derived.by(() => {
		const match = page.url.pathname.match(SITE_ROUTE);
		if (!match) return null;
		const id = match[1];
		if (id === 'new') return null;
		return id ?? null;
	});

	// Topbar reads the active site id; route takes precedence, falling back
	// to whatever Worker E's site loader has set in currentSite.
	const activeSiteId = $derived(siteIdFromRoute ?? currentSite.value?.id ?? null);
</script>

<div class="min-h-[100dvh] bg-bg text-text-primary">
	<Topbar currentSiteId={activeSiteId} />
	<main class="mx-auto max-w-7xl px-6 py-6">
		{@render children()}
	</main>
</div>

<CommandPalette />

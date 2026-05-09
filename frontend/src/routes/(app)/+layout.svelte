<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
	import Topbar from '$lib/components/layout/Topbar.svelte';
	import CommandPalette from '$lib/components/ui/CommandPalette.svelte';
	import ConfirmDialog from '$lib/components/ui/ConfirmDialog.svelte';
	import { currentSite } from '$lib/stores/currentSite.svelte';
	import { auth } from '$lib/stores/auth.svelte';
	import { ShieldAlert } from 'lucide-svelte';

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

	// Hide the MFA banner on the /account page itself; the user is
	// already in the right place to enroll.
	const onAccountPage = $derived(page.url.pathname.startsWith('/account'));
	const showMfaBanner = $derived(auth.enrollRequired && !onAccountPage);
</script>

<div class="min-h-[100dvh] bg-bg text-text-primary">
	<Topbar currentSiteId={activeSiteId} />
	{#if showMfaBanner}
		<div
			role="alert"
			class="border-b border-warning/40 bg-warning/10 px-6 py-2.5 text-[13px] text-text-primary"
		>
			<div class="mx-auto flex max-w-7xl items-center gap-3">
				<ShieldAlert size={16} strokeWidth={1.75} class="shrink-0 text-warning" />
				<p class="flex-1">
					Two-factor authentication enrollment is required by your workspace
					policy. Writes are blocked until you enroll.
				</p>
				<a
					href="/account#mfa"
					class="inline-flex h-7 items-center gap-1.5 rounded-md border border-warning/50 bg-bg-surface px-3 text-[12px] font-medium text-text-primary hover:bg-warning/10"
				>
					Enroll now
				</a>
			</div>
		</div>
	{/if}
	<main class="mx-auto max-w-7xl px-6 py-6">
		{@render children()}
	</main>
</div>

<CommandPalette />
<ConfirmDialog />

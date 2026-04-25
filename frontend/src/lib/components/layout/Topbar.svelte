<script lang="ts">
	import KeyboardHint from '$lib/components/ui/KeyboardHint.svelte';
	import ThemeToggle from '$lib/components/ui/ThemeToggle.svelte';
	import SiteSwitcher from './SiteSwitcher.svelte';
	import UserMenu from './UserMenu.svelte';
	import { openPalette } from '$lib/stores/commandPalette.svelte';

	let {
		currentSiteId = null
	}: {
		currentSiteId?: string | null;
	} = $props();
</script>

<header
	class="sticky top-0 z-30 h-14 border-b border-border/50 bg-bg-surface/85 backdrop-blur"
>
	<div class="mx-auto flex h-full max-w-7xl items-center justify-between gap-4 px-6">
		<div class="flex shrink-0 items-center">
			<SiteSwitcher {currentSiteId} />
		</div>

		<div class="flex shrink-0 items-center gap-1">
			<button
				type="button"
				onclick={() => openPalette()}
				class="inline-flex h-9 items-center gap-2 rounded-md px-2.5 text-[12px] text-text-muted transition-colors hover:bg-bg-hover hover:text-text-secondary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg"
				aria-label="Open command palette"
			>
				<span class="hidden sm:inline">Search</span>
				<KeyboardHint keys={["⌘", 'K']} />
			</button>
			<ThemeToggle />
			<UserMenu />
		</div>
	</div>
</header>

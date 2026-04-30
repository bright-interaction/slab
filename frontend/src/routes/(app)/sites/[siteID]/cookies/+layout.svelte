<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
	import type { Site } from '$lib/api/types';

	let {
		data,
		children
	}: {
		data: { site: Site };
		children: Snippet;
	} = $props();

	type Tab = { id: string; label: string; href: string };

	const tabs = $derived<Tab[]>([
		{ id: 'overview', label: 'Settings', href: `/sites/${data.site.id}/cookies` },
		{ id: 'proofs', label: 'Proofs', href: `/sites/${data.site.id}/cookies/proofs` },
		{ id: 'analytics', label: 'Analytics', href: `/sites/${data.site.id}/cookies/analytics` }
	]);

	function isActive(href: string): boolean {
		const path = page.url.pathname;
		if (href === `/sites/${data.site.id}/cookies`) {
			return path === href;
		}
		return path === href || path.startsWith(`${href}/`);
	}
</script>

<div class="border-b border-border-light bg-bg-surface/40">
	<div class="mx-auto flex max-w-7xl items-center gap-1 px-6">
		{#each tabs as tab (tab.id)}
			{@const active = isActive(tab.href)}
			<a
				href={tab.href}
				class="relative -mb-px inline-flex items-center px-3 py-2.5 text-[12px] transition-colors focus-visible:outline-none focus-visible:text-text-primary {active
					? 'text-text-primary border-b-2 border-accent font-medium'
					: 'text-text-muted hover:text-text-primary border-b-2 border-transparent'}"
				aria-current={active ? 'page' : undefined}
			>
				{tab.label}
			</a>
		{/each}
	</div>
</div>

{@render children()}

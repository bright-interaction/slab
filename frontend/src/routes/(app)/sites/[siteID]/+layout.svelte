<script lang="ts">
	import type { Snippet } from 'svelte';
	import SectionNav from '$lib/components/layout/SectionNav.svelte';
	import type { Site } from '$lib/api/types';

	let {
		data,
		children
	}: {
		data: { site: Site };
		children: Snippet;
	} = $props();

	const swatches = $derived(
		[
			{ key: 'primary', value: data.site.primary_color },
			{ key: 'secondary', value: data.site.secondary_color },
			{ key: 'bg', value: data.site.bg_color },
			{ key: 'text', value: data.site.text_color }
		].filter((s) => s.value)
	);
</script>

<header class="border-b border-border-light bg-bg">
	<div class="mx-auto flex max-w-7xl items-center justify-between px-6 py-5">
		<div class="flex items-baseline gap-3">
			<h1 class="font-display text-2xl font-extralight tracking-tight text-text-primary">
				{data.site.name}
			</h1>
			{#if data.site.domain}
				<span class="font-mono text-xs text-text-muted">{data.site.domain}</span>
			{/if}
		</div>
		{#if swatches.length > 0}
			<div class="flex items-center gap-1.5" aria-label="Brand colors">
				{#each swatches as swatch (swatch.key)}
					<span
						class="h-3 w-3 rounded-full ring-1 ring-border-light"
						style="background-color: {swatch.value};"
						title={`${swatch.key}: ${swatch.value}`}
					></span>
				{/each}
			</div>
		{/if}
	</div>
</header>

<SectionNav siteID={data.site.id} />

{@render children()}

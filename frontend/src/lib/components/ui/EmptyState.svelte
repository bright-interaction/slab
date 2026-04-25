<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		description,
		icon,
		action,
		class: className = ''
	}: {
		title: string;
		description?: string;
		icon?: Snippet;
		action?: Snippet;
		class?: string;
	} = $props();
</script>

<div class="flex flex-col items-center justify-center py-16 px-6 text-center {className}">
	<div class="mb-4 flex h-14 w-14 items-center justify-center rounded-full bg-bg-elevated text-text-muted">
		{#if icon}
			{@render icon()}
		{:else}
			<svg
				class="h-6 w-6"
				fill="none"
				viewBox="0 0 24 24"
				stroke-width="1.5"
				stroke="currentColor"
			>
				<path
					stroke-linecap="round"
					stroke-linejoin="round"
					d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M9.75 11.25l1.5 1.5 3-3"
				/>
			</svg>
		{/if}
	</div>
	<h3 class="font-display text-lg font-extralight tracking-tight text-text-primary">{title}</h3>
	{#if description}
		<p class="mt-1 max-w-sm text-[13px] text-text-secondary">{description}</p>
	{/if}
	{#if action}
		<div class="mt-5">
			{@render action()}
		</div>
	{/if}
</div>

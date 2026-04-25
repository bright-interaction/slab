<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		padding = 'md',
		hoverable = false,
		class: className = '',
		children
	}: {
		padding?: 'none' | 'sm' | 'md' | 'lg';
		hoverable?: boolean;
		class?: string;
		children: Snippet;
	} = $props();

	const padClasses: Record<string, string> = {
		none: '',
		sm: 'p-3',
		md: 'p-5',
		lg: 'p-7'
	};
</script>

<div
	class="rounded-xl bg-bg-surface border border-border-light {padClasses[padding]} {hoverable
		? 'transition-colors hover:bg-bg-hover cursor-pointer'
		: ''} {className}"
>
	{@render children()}
</div>

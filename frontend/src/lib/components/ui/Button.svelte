<script lang="ts">
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	let {
		variant = 'primary',
		size = 'md',
		loading = false,
		disabled = false,
		type = 'button',
		icon,
		children,
		class: className = '',
		...rest
	}: HTMLButtonAttributes & {
		variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
		size?: 'sm' | 'md' | 'lg';
		loading?: boolean;
		icon?: Snippet;
		children?: Snippet;
		class?: string;
	} = $props();

	const variantClasses: Record<string, string> = {
		primary: 'bg-accent text-accent-fg hover:bg-accent-hover font-medium',
		secondary:
			'bg-bg-elevated border border-border text-text-primary hover:bg-bg-hover font-medium',
		ghost: 'text-text-secondary hover:text-text-primary hover:bg-bg-hover',
		danger: 'bg-danger/10 text-danger hover:bg-danger/20 border border-danger/40 font-medium'
	};

	const sizeClasses: Record<string, string> = {
		sm: 'h-8 px-3 text-xs gap-1.5',
		md: 'h-9 px-4 text-[13px] gap-2',
		lg: 'h-10 px-5 text-sm gap-2'
	};
</script>

<button
	{type}
	class="inline-flex items-center justify-center rounded-lg transition-all duration-200 active:scale-[0.98] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg disabled:pointer-events-none disabled:opacity-50 {variantClasses[
		variant
	]} {sizeClasses[size]} {className}"
	disabled={disabled || loading}
	{...rest}
>
	{#if loading}
		<svg class="h-4 w-4 animate-spin shrink-0" viewBox="0 0 24 24" fill="none">
			<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"
			></circle>
			<path
				class="opacity-75"
				fill="currentColor"
				d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
			></path>
		</svg>
	{:else if icon}
		<span class="shrink-0">{@render icon()}</span>
	{/if}
	{#if children}
		{@render children()}
	{/if}
</button>

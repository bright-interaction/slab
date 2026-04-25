<script lang="ts">
	type Variant = 'default' | 'success' | 'warning' | 'danger' | 'info' | 'accent';

	let {
		variant = 'default',
		dot = false,
		class: className = '',
		children
	}: {
		variant?: Variant;
		dot?: boolean;
		class?: string;
		children: import('svelte').Snippet;
	} = $props();

	const variants: Record<Variant, string> = {
		default: 'bg-bg-elevated text-text-secondary border border-border',
		success: 'bg-success/10 text-success border border-success/20',
		warning: 'bg-warning/10 text-warning border border-warning/20',
		danger: 'bg-danger/10 text-danger border border-danger/20',
		info: 'bg-info/10 text-info border border-info/20',
		accent: 'bg-accent text-accent-fg border border-transparent'
	};

	const dotColors: Record<Variant, string> = {
		default: 'bg-text-muted',
		success: 'bg-success',
		warning: 'bg-warning',
		danger: 'bg-danger',
		info: 'bg-info',
		accent: 'bg-accent-fg'
	};
</script>

<span
	class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-medium whitespace-nowrap {variants[
		variant
	]} {className}"
>
	{#if dot}
		<span class="h-1.5 w-1.5 rounded-full {dotColors[variant]}"></span>
	{/if}
	{@render children()}
</span>

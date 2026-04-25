<script lang="ts">
	let {
		size = 'md',
		class: className = ''
	}: { size?: 'sm' | 'md' | 'lg'; class?: string } = $props();

	const dim: Record<string, number> = {
		sm: 14,
		md: 18,
		lg: 24
	};
</script>

<svg
	class="animate-spin text-text-muted {className}"
	width={dim[size]}
	height={dim[size]}
	viewBox="0 0 24 24"
	fill="none"
	role="status"
	aria-label="Loading"
>
	<circle class="opacity-20" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3"></circle>
	<path
		class="opacity-80"
		fill="currentColor"
		d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
	></path>
</svg>

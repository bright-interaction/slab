<script lang="ts">
	let {
		width = '100%',
		height = '1rem',
		rounded = 'default',
		class: className = ''
	}: {
		width?: string | number;
		height?: string | number;
		rounded?: 'default' | 'full';
		class?: string;
	} = $props();

	const w = $derived(typeof width === 'number' ? `${width}px` : width);
	const h = $derived(typeof height === 'number' ? `${height}px` : height);
</script>

<span
	class="block overflow-hidden bg-bg-hover {rounded === 'full' ? 'rounded-full' : 'rounded-md'} {className}"
	style="width: {w}; height: {h};"
	aria-hidden="true"
>
	<span class="block h-full w-full skeleton-shimmer"></span>
</span>

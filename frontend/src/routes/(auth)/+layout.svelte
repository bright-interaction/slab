<script lang="ts">
	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();
</script>

<div class="min-h-[100dvh] flex items-center justify-center bg-bg px-6">
	<div class="w-full max-w-sm">
		<div class="mb-8 text-center">
			<p class="font-mono text-[11px] uppercase tracking-[0.2em] text-text-muted">slab</p>
		</div>
		{@render children()}
	</div>
</div>

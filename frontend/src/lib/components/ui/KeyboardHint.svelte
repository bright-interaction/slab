<script lang="ts">
	let {
		keys,
		class: className = ''
	}: {
		keys: string[];
		class?: string;
	} = $props();
</script>

<span class="inline-flex items-center gap-1 {className}">
	{#each keys as k, i (i)}
		<kbd
			class="inline-flex h-5 min-w-[1.25rem] items-center justify-center rounded border border-border bg-bg-elevated px-1 font-mono text-[10px] font-medium text-text-muted tracking-wide"
		>
			{k}
		</kbd>
	{/each}
</span>

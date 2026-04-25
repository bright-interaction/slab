<script lang="ts">
	import { Checkbox as BitsCheckbox } from 'bits-ui';

	let {
		checked = $bindable(false),
		disabled = false,
		id,
		label,
		class: className = ''
	}: {
		checked?: boolean;
		disabled?: boolean;
		id?: string;
		label?: string;
		class?: string;
	} = $props();
</script>

<div class="inline-flex items-center gap-2 {className}">
	<BitsCheckbox.Root
		bind:checked
		{disabled}
		{id}
		class="flex h-4 w-4 items-center justify-center rounded border-[1.5px] border-border bg-bg-elevated transition-colors data-[state=checked]:bg-accent data-[state=checked]:border-accent hover:border-accent/60 disabled:cursor-not-allowed disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg"
	>
		{#snippet children({ checked: isChecked })}
			{#if isChecked}
				<svg
					class="h-3 w-3 text-accent-fg"
					fill="none"
					viewBox="0 0 24 24"
					stroke-width="3"
					stroke="currentColor"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
				</svg>
			{/if}
		{/snippet}
	</BitsCheckbox.Root>
	{#if label}
		<label for={id} class="text-[13px] text-text-primary cursor-pointer select-none">{label}</label>
	{/if}
</div>

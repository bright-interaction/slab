<script lang="ts">
	import { Switch as BitsSwitch } from 'bits-ui';

	let {
		checked = $bindable(false),
		disabled = false,
		id,
		label,
		ariaLabel,
		class: className = ''
	}: {
		checked?: boolean;
		disabled?: boolean;
		id?: string;
		label?: string;
		ariaLabel?: string;
		class?: string;
	} = $props();
</script>

<div class="inline-flex items-center gap-2 {className}">
	<BitsSwitch.Root
		bind:checked
		{disabled}
		{id}
		aria-label={ariaLabel ?? label}
		class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border border-transparent transition-colors data-[state=checked]:bg-accent data-[state=unchecked]:bg-bg-hover disabled:cursor-not-allowed disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg"
	>
		<BitsSwitch.Thumb
			class="pointer-events-none block h-3.5 w-3.5 rounded-full bg-bg-surface shadow-sm transition-transform data-[state=checked]:translate-x-[18px] data-[state=unchecked]:translate-x-[3px]"
		/>
	</BitsSwitch.Root>
	{#if label}
		<label for={id} class="text-[13px] text-text-primary cursor-pointer select-none">{label}</label>
	{/if}
</div>

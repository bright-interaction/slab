<script lang="ts">
	import { WIZARD_STEPS, type WizardStep } from '$lib/stores/wizard.svelte';

	let { current }: { current: WizardStep } = $props();

	const labels: Record<WizardStep, string> = {
		type: 'Type',
		info: 'Info',
		structure: 'Structure',
		branding: 'Branding',
		confirm: 'Review'
	};

	const currentIndex = $derived(WIZARD_STEPS.indexOf(current));
</script>

<nav aria-label="Wizard progress" class="w-full">
	<ol class="flex items-center justify-between gap-2">
		{#each WIZARD_STEPS as step, idx (step)}
			{@const isActive = idx === currentIndex}
			{@const isComplete = idx < currentIndex}
			<li class="flex flex-1 items-center gap-2 last:flex-none">
				<div class="flex items-center gap-2 shrink-0">
					<span
						class="flex h-6 w-6 items-center justify-center rounded-full border text-[11px] font-medium transition-colors {isActive
							? 'border-accent bg-accent text-accent-fg'
							: isComplete
								? 'border-accent/40 bg-accent/10 text-accent'
								: 'border-border bg-bg-elevated text-text-muted'}"
						aria-current={isActive ? 'step' : undefined}
					>
						{idx + 1}
					</span>
					<span
						class="text-[12px] font-medium {isActive
							? 'text-text-primary'
							: isComplete
								? 'text-text-secondary'
								: 'text-text-muted'}"
					>
						{labels[step]}
					</span>
				</div>
				{#if idx < WIZARD_STEPS.length - 1}
					<span
						class="h-px flex-1 transition-colors {isComplete
							? 'bg-accent/40'
							: 'bg-border'}"
					></span>
				{/if}
			</li>
		{/each}
	</ol>
</nav>

<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import WizardProgress from '$lib/components/wizard/WizardProgress.svelte';
	import ConfirmDialog, { confirm } from '$lib/components/ui/ConfirmDialog.svelte';
	import { reset, WIZARD_STEPS, type WizardStep } from '$lib/stores/wizard.svelte';

	let { children } = $props();

	const currentStep = $derived.by<WizardStep>(() => {
		const segment = page.params.step as WizardStep | undefined;
		if (segment && WIZARD_STEPS.includes(segment)) return segment;
		return 'type';
	});

	async function handleEscape(): Promise<void> {
		const ok = await confirm({
			title: 'Discard wizard progress?',
			message:
				'You will lose the selections you have made. You can start a new site any time.',
			confirmText: 'Discard',
			cancelText: 'Keep editing',
			variant: 'danger'
		});
		if (ok) {
			reset();
			void goto('/sites');
		}
	}

	onMount(() => {
		const handler = (e: KeyboardEvent): void => {
			if (e.key !== 'Escape') return;
			const target = e.target as HTMLElement | null;
			if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA')) {
				return;
			}
			e.preventDefault();
			void handleEscape();
		};
		window.addEventListener('keydown', handler);
		return () => window.removeEventListener('keydown', handler);
	});
</script>

<div class="mx-auto flex min-h-screen w-full max-w-5xl flex-col gap-8 px-6 py-10">
	<header class="flex flex-col gap-4">
		<div class="flex items-center justify-between">
			<div class="flex flex-col">
				<span class="text-[11px] font-medium uppercase tracking-wide text-text-muted">
					New site
				</span>
				<h1
					class="font-display text-3xl font-extralight tracking-tight text-text-primary"
				>
					Onboarding wizard
				</h1>
			</div>
			<button
				type="button"
				class="text-[12px] text-text-muted transition-colors hover:text-text-primary"
				onclick={() => void handleEscape()}
			>
				Cancel
			</button>
		</div>
		<WizardProgress current={currentStep} />
	</header>

	<main class="flex-1">
		{@render children()}
	</main>
</div>

<ConfirmDialog />

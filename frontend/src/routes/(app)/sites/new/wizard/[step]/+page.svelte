<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import Button from '$lib/components/ui/Button.svelte';
	import StepType from '$lib/components/wizard/StepType.svelte';
	import StepKit from '$lib/components/wizard/StepKit.svelte';
	import StepInfo from '$lib/components/wizard/StepInfo.svelte';
	import StepStructure from '$lib/components/wizard/StepStructure.svelte';
	import StepBranding from '$lib/components/wizard/StepBranding.svelte';
	import StepConfirm from '$lib/components/wizard/StepConfirm.svelte';
	import {
		WIZARD_STEPS,
		setStep,
		wizard,
		type WizardStep
	} from '$lib/stores/wizard.svelte';

	const SLUG_RE = /^[a-z0-9-]+$/;
	const EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
	const HEX_RE = /^#[0-9a-fA-F]{6}$/;

	const currentStep = $derived.by<WizardStep>(() => {
		const segment = page.params.step as WizardStep | undefined;
		if (segment && WIZARD_STEPS.includes(segment)) return segment;
		return 'type';
	});

	onMount(() => {
		const segment = page.params.step;
		if (!segment || !WIZARD_STEPS.includes(segment as WizardStep)) {
			void goto('/sites/new/wizard/type', { replaceState: true });
			return;
		}
		setStep(segment as WizardStep);
	});

	$effect(() => {
		if (WIZARD_STEPS.includes(currentStep)) {
			setStep(currentStep);
		}
	});

	const currentIndex = $derived(WIZARD_STEPS.indexOf(currentStep));

	const validation = $derived.by(() => {
		const ws = wizard.value;
		switch (currentStep) {
			case 'type':
				return { valid: ws.type !== null, message: 'Pick a site type to continue.' };
			case 'kit':
				return { valid: true, message: '' };
			case 'info': {
				const info = ws.info;
				if (info.name.trim().length < 2) {
					return { valid: false, message: 'Name must be at least 2 characters.' };
				}
				if (!SLUG_RE.test(info.slug) || info.slug.length === 0 || info.slug.length > 50) {
					return { valid: false, message: 'Slug must match a-z, 0-9, hyphens (max 50).' };
				}
				if (!EMAIL_RE.test(info.contact_email.trim())) {
					return { valid: false, message: 'Enter a valid contact email.' };
				}
				if (info.business_name.trim().length === 0) {
					return { valid: false, message: 'Business name is required.' };
				}
				return { valid: true, message: '' };
			}
			case 'structure': {
				const s = ws.structure;
				if (s.structure_type === 'one-pager') return { valid: true, message: '' };
				if (ws.silos.length === 0) {
					return { valid: false, message: 'Add at least one silo or pick one-pager.' };
				}
				for (const silo of ws.silos) {
					if (silo.name.trim().length === 0 || !SLUG_RE.test(silo.slug_prefix)) {
						return { valid: false, message: 'Each silo needs a name and valid slug prefix.' };
					}
				}
				return { valid: true, message: '' };
			}
			case 'branding': {
				const b = ws.branding;
				const colors = [b.primary_color, b.secondary_color, b.bg_color, b.text_color];
				for (const c of colors) {
					if (!HEX_RE.test(c)) {
						return { valid: false, message: 'Colors must be 6-digit hex (e.g. #0e7490).' };
					}
				}
				return { valid: true, message: '' };
			}
			case 'confirm':
				return { valid: true, message: '' };
		}
	});

	function goBack(): void {
		if (currentIndex <= 0) return;
		const prev = WIZARD_STEPS[currentIndex - 1];
		void goto(`/sites/new/wizard/${prev}`);
	}

	function goNext(): void {
		if (!validation.valid) return;
		if (currentIndex >= WIZARD_STEPS.length - 1) return;
		const next = WIZARD_STEPS[currentIndex + 1];
		void goto(`/sites/new/wizard/${next}`);
	}
</script>

<div class="flex flex-col gap-8">
	<section class="animate-fadeIn">
		{#if currentStep === 'type'}
			<StepType />
		{:else if currentStep === 'kit'}
			<StepKit />
		{:else if currentStep === 'info'}
			<StepInfo />
		{:else if currentStep === 'structure'}
			<StepStructure />
		{:else if currentStep === 'branding'}
			<StepBranding />
		{:else if currentStep === 'confirm'}
			<StepConfirm />
		{/if}
	</section>

	{#if currentStep !== 'confirm'}
		<div class="flex items-center justify-between border-t border-border-light pt-6">
			<Button variant="secondary" disabled={currentIndex === 0} onclick={goBack}>
				Back
			</Button>
			<div class="flex items-center gap-3">
				{#if !validation.valid && validation.message}
					<span class="text-[12px] text-text-muted">{validation.message}</span>
				{/if}
				<Button variant="primary" disabled={!validation.valid} onclick={goNext}>
					Next
				</Button>
			</div>
		</div>
	{:else}
		<div class="flex items-center justify-between border-t border-border-light pt-6">
			<Button variant="secondary" onclick={goBack}>Back</Button>
		</div>
	{/if}
</div>

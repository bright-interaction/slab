<script lang="ts">
	import { goto } from '$app/navigation';
	import Button from '$lib/components/ui/Button.svelte';
	import * as sitesApi from '$lib/api/sites';
	import { ApiError } from '$lib/api/client';
	import { reset, submit, wizard } from '$lib/stores/wizard.svelte';
	import { toast } from '$lib/stores/toast.svelte';

	const ws = $derived(wizard.value);

	const seedItems = [
		'Privacy policy, terms, cookie policy, and 404 pages',
		'Default knowledgebase entries with your tone and audience',
		'Default guardrails for safe agent edits',
		'Header and footer global blocks',
		'Security defaults: CSP, headers, robots.txt, llms.txt'
	];

	let submitting = $state(false);
	let errorMessage = $state<string | null>(null);

	async function handleSubmit(): Promise<void> {
		errorMessage = null;
		submitting = true;
		try {
			const res = await submit({ seed: sitesApi.seed });
			reset();
			toast.success('Site created');
			void goto(`/sites/${res.site_id}`);
		} catch (err) {
			submitting = false;
			if (err instanceof ApiError) {
				if (err.status === 409) {
					errorMessage =
						'A site with this slug already exists. Go back to step 2 to change it.';
					return;
				}
				if (err.status === 400) {
					errorMessage = err.message || 'Validation failed. Review the previous steps.';
					return;
				}
				errorMessage = err.message || 'Failed to create site.';
				return;
			}
			errorMessage = 'Failed to create site. Please try again.';
		}
	}

	function summaryRow(label: string, value: string): { label: string; value: string } {
		return { label, value: value || 'not set' };
	}

	const summaryRows = $derived([
		summaryRow('Type', ws.type ?? 'not set'),
		summaryRow('Site name', ws.info.name),
		summaryRow('Slug', ws.info.slug),
		summaryRow('Domain', ws.info.domain || 'not set'),
		summaryRow('Language', ws.info.lang),
		summaryRow('Country', ws.info.country),
		summaryRow('Business name', ws.info.business_name),
		summaryRow('Contact email', ws.info.contact_email),
		summaryRow('Structure', ws.structure.structure_type),
		summaryRow('Max depth', String(ws.structure.max_depth))
	]);
</script>

<div class="flex flex-col gap-6">
	<div class="flex flex-col gap-1">
		<h2 class="font-display text-2xl font-extralight tracking-tight text-text-primary">
			Review and create
		</h2>
		<p class="text-[13px] text-text-secondary">
			Confirm your selections. We'll seed everything below in a single transaction.
		</p>
	</div>

	<div class="grid grid-cols-1 gap-4 lg:grid-cols-[1.2fr_1fr]">
		<div class="flex flex-col gap-3 rounded-xl border border-border-light bg-bg-surface p-5">
			<span class="text-[12px] font-medium text-text-secondary">Your selections</span>
			<dl class="grid grid-cols-1 gap-y-2 sm:grid-cols-[140px_1fr]">
				{#each summaryRows as row (row.label)}
					<dt class="text-[12px] text-text-muted">{row.label}</dt>
					<dd class="text-[13px] text-text-primary">{row.value}</dd>
				{/each}
				{#if ws.silos.length > 0}
					<dt class="text-[12px] text-text-muted">Silos</dt>
					<dd class="flex flex-col gap-1 text-[13px] text-text-primary">
						{#each ws.silos as silo, idx (idx)}
							<span class="font-mono text-[12px]">/{silo.slug_prefix}</span>
						{/each}
					</dd>
				{/if}
				<dt class="text-[12px] text-text-muted">Colors</dt>
				<dd class="flex items-center gap-1.5">
					{#each [ws.branding.primary_color, ws.branding.secondary_color, ws.branding.bg_color, ws.branding.text_color] as c (c)}
						<span
							class="h-5 w-5 rounded-md border border-border-light"
							style="background:{c};"
							title={c}
						></span>
					{/each}
				</dd>
				<dt class="text-[12px] text-text-muted">Fonts</dt>
				<dd class="text-[13px] text-text-primary">
					{ws.branding.font_heading} / {ws.branding.font_body}
				</dd>
			</dl>
		</div>

		<div class="flex flex-col gap-3 rounded-xl border border-border-light bg-bg-surface p-5">
			<span class="text-[12px] font-medium text-text-secondary">We'll create</span>
			<ul class="flex flex-col gap-2">
				{#each seedItems as item (item)}
					<li class="flex items-start gap-2 text-[13px] text-text-primary">
						<svg
							class="mt-0.5 h-4 w-4 shrink-0 text-accent"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="2"
							stroke="currentColor"
						>
							<path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
						</svg>
						<span>{item}</span>
					</li>
				{/each}
			</ul>
		</div>
	</div>

	{#if errorMessage}
		<div
			class="rounded-xl border border-danger/30 bg-danger/10 px-4 py-3 text-[13px] text-danger"
			role="alert"
		>
			{errorMessage}
		</div>
	{/if}

	<div class="flex justify-end">
		<Button variant="primary" size="lg" loading={submitting} onclick={handleSubmit}>
			Create site
		</Button>
	</div>
</div>

<script lang="ts">
	import { ChevronDown } from 'lucide-svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import CategoryDonut from './CategoryDonut.svelte';
	import CheckRow from './CheckRow.svelte';
	import * as evaluationsApi from '$lib/api/evaluations';
	import type { Evaluation, EvaluationCheck } from '$lib/api/types';

	let {
		category,
		evaluation,
		siteID
	}: {
		category: string;
		evaluation: Evaluation | null;
		siteID: string;
	} = $props();

	const checks = $derived<EvaluationCheck[]>(
		evaluation ? evaluationsApi.parseEvaluationChecks(evaluation.checks_json) : []
	);

	const failedChecks = $derived(checks.filter((c) => c.passed !== true));
	const passedChecks = $derived(checks.filter((c) => c.passed === true));
	const total = $derived(
		checks.filter((c) => c.passed !== undefined && c.passed !== null).length
	);
	const passedCount = $derived(passedChecks.length);

	// Failures auto-show on first render so the user sees the explanation
	// without an extra click. Passing checks are hidden behind a toggle to
	// keep the card scannable on a healthy A+ result.
	let showFailing = $state(true);
	let showPassing = $state(false);

	const hasFailing = $derived(failedChecks.length > 0);
	const hasPassing = $derived(passedChecks.length > 0);

	function toggleFailing() {
		showFailing = !showFailing;
	}
	function togglePassing() {
		showPassing = !showPassing;
	}

	const categoryBlurb: Record<string, string> = {
		security: 'Security headers, CSP quality, SRI, mixed content, security.txt.',
		seo: 'Meta tags, headings, schema, robots, sitemap, llms.txt, GEO signals.',
		performance: 'HTML size, render-blocking scripts, image formats, lazy loading.',
		accessibility: 'Landmarks, alt text, form labels, contrast, focus indicators.',
		privacy: 'Consent, trackers, AI-bot blocking, cookie hygiene.'
	};
</script>

<Card padding="md">
	<div class="flex flex-col gap-5 sm:flex-row sm:items-center">
		<!-- Donut + label -->
		<div class="shrink-0">
			<CategoryDonut {category} {evaluation} size="sm" />
		</div>

		<!-- Headline / score / blurb -->
		<div class="min-w-0 flex-1">
			<p class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				{category}
			</p>
			<h3
				class="mt-1 font-display text-2xl font-extralight tracking-tight text-text-primary capitalize"
			>
				{category}
			</h3>
			<p class="mt-1 text-[12px] text-text-muted">
				{categoryBlurb[category] ?? ''}
			</p>
			{#if evaluation}
				<p class="mt-3 text-[13px] text-text-secondary">
					<span class="font-medium text-text-primary"
						>{evaluation.score} / {evaluation.max_score}</span
					>
					<span class="text-text-muted">·</span>
					{passedCount} of {total} checks passed
					{#if hasFailing}
						<span class="text-text-muted">·</span>
						<span class="text-warning">{failedChecks.length} need attention</span>
					{/if}
				</p>
			{:else}
				<p class="mt-3 text-[12px] text-text-muted">No data for this category.</p>
			{/if}
		</div>
	</div>

	<!-- Failing checks section (auto-expanded by default). -->
	{#if hasFailing}
		<div class="mt-5 border-t border-border-light pt-5">
			<button
				type="button"
				onclick={toggleFailing}
				class="flex w-full items-center justify-between text-left"
				aria-expanded={showFailing}
			>
				<div>
					<p class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Needs attention
					</p>
					<p class="mt-0.5 text-[12px] text-text-secondary">
						{failedChecks.length} check{failedChecks.length === 1 ? '' : 's'} not passing.
						Each row links to the fix.
					</p>
				</div>
				<ChevronDown
					class="h-4 w-4 shrink-0 text-text-muted transition-transform {showFailing
						? 'rotate-180'
						: ''}"
				/>
			</button>
			{#if showFailing}
				<div class="mt-3 grid grid-cols-1 gap-2 lg:grid-cols-2">
					{#each failedChecks as check, i (check.id ?? `${check.name}-${i}`)}
						<CheckRow {check} {siteID} />
					{/each}
				</div>
			{/if}
		</div>
	{/if}

	<!-- Passing checks section (collapsed by default). -->
	{#if hasPassing}
		<div class="mt-5 border-t border-border-light pt-5">
			<button
				type="button"
				onclick={togglePassing}
				class="flex w-full items-center justify-between text-left"
				aria-expanded={showPassing}
			>
				<div>
					<p class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Passing
					</p>
					<p class="mt-0.5 text-[12px] text-text-secondary">
						{passedChecks.length} check{passedChecks.length === 1 ? '' : 's'} look good.
					</p>
				</div>
				<ChevronDown
					class="h-4 w-4 shrink-0 text-text-muted transition-transform {showPassing
						? 'rotate-180'
						: ''}"
				/>
			</button>
			{#if showPassing}
				<div class="mt-3 grid grid-cols-1 gap-2 lg:grid-cols-2">
					{#each passedChecks as check, i (check.id ?? `${check.name}-${i}`)}
						<CheckRow {check} {siteID} />
					{/each}
				</div>
			{/if}
		</div>
	{/if}
</Card>

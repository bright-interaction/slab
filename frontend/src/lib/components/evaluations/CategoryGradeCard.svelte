<script lang="ts">
	import { ChevronDown } from 'lucide-svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import GradeBadge from '$lib/components/ui/GradeBadge.svelte';
	import CheckRow from './CheckRow.svelte';
	import * as evaluationsApi from '$lib/api/evaluations';
	import type { Evaluation, EvaluationCheck } from '$lib/api/types';

	let {
		category,
		evaluation,
		siteID,
		defaultExpanded = false
	}: {
		category: string;
		evaluation: Evaluation | null;
		siteID: string;
		defaultExpanded?: boolean;
	} = $props();

	// eslint-disable-next-line svelte/no-state-reference-loop -- intended: prop seeds initial state
	let expanded = $state<boolean>(defaultExpanded === true);

	type Grade = 'A+' | 'A' | 'B+' | 'B' | 'C' | 'D' | 'F';
	const validGrades: Grade[] = ['A+', 'A', 'B+', 'B', 'C', 'D', 'F'];
	function asGrade(value: string | undefined): Grade | null {
		if (!value) return null;
		return (validGrades as string[]).includes(value) ? (value as Grade) : null;
	}

	const checks = $derived<EvaluationCheck[]>(
		evaluation ? evaluationsApi.parseEvaluationChecks(evaluation.checks_json) : []
	);

	const passed = $derived(checks.filter((c) => c.passed === true).length);
	const total = $derived(checks.filter((c) => c.passed !== undefined && c.passed !== null).length);
	const grade = $derived(asGrade(evaluation?.grade));

	function toggle() {
		expanded = !expanded;
	}
</script>

<Card padding="md">
	<div class="flex items-start justify-between gap-4">
		<div class="min-w-0">
			<p class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				{category}
			</p>
			<h3 class="mt-1 font-display text-2xl font-extralight tracking-tight text-text-primary capitalize">
				{category}
			</h3>
			{#if evaluation}
				<p class="mt-2 text-[13px] text-text-secondary">
					{evaluation.score} / {evaluation.max_score}
				</p>
				<p class="mt-0.5 text-[12px] text-text-muted">
					{passed} of {total} checks passed
				</p>
			{:else}
				<p class="mt-2 text-[12px] text-text-muted">No data for this category.</p>
			{/if}
		</div>
		<div class="shrink-0">
			{#if grade}
				<GradeBadge {grade} size="md" />
			{:else}
				<span
					class="inline-flex h-14 w-14 items-center justify-center rounded-2xl bg-bg-elevated text-text-muted text-2xl"
				>
					?
				</span>
			{/if}
		</div>
	</div>

	{#if checks.length > 0}
		<button
			type="button"
			onclick={toggle}
			class="mt-4 inline-flex items-center gap-1.5 text-[12px] text-text-secondary hover:text-text-primary transition-colors"
			aria-expanded={expanded}
		>
			{expanded ? 'Hide' : 'Show'} all checks
			<ChevronDown
				class="h-3.5 w-3.5 transition-transform {expanded ? 'rotate-180' : ''}"
			/>
		</button>

		{#if expanded}
			<div class="mt-3 space-y-2">
				{#each checks as check, i (check.id ?? `${check.name}-${i}`)}
					<CheckRow {check} {siteID} />
				{/each}
			</div>
		{/if}
	{/if}
</Card>

<script lang="ts">
	import type { Evaluation } from '$lib/api/types';

	let {
		category,
		evaluation
	}: {
		category: string;
		evaluation: Evaluation | null;
	} = $props();

	type Grade = 'A+' | 'A' | 'B+' | 'B' | 'C' | 'D' | 'F';
	const validGrades: Grade[] = ['A+', 'A', 'B+', 'B', 'C', 'D', 'F'];
	function asGrade(value: string | undefined): Grade | null {
		if (!value) return null;
		return (validGrades as string[]).includes(value) ? (value as Grade) : null;
	}

	const grade = $derived(asGrade(evaluation?.grade));

	const pct = $derived.by<number>(() => {
		if (!evaluation || evaluation.max_score <= 0) return 0;
		const v = (evaluation.score / evaluation.max_score) * 100;
		if (Number.isNaN(v)) return 0;
		return Math.max(0, Math.min(100, v));
	});

	const tone = $derived.by<'success' | 'warning' | 'danger' | 'muted'>(() => {
		if (!grade) return 'muted';
		if (grade === 'A+' || grade === 'A' || grade === 'B+' || grade === 'B') return 'success';
		if (grade === 'C') return 'warning';
		return 'danger';
	});

	const ringColor = $derived(
		tone === 'success'
			? 'var(--t-success)'
			: tone === 'warning'
				? 'var(--t-warning)'
				: tone === 'danger'
					? 'var(--t-danger)'
					: 'var(--t-text-muted)'
	);

	const radius = 42;
	const circ = 2 * Math.PI * radius;
	const dashOffset = $derived(circ * (1 - pct / 100));
</script>

<div class="flex flex-col items-center gap-2">
	<div class="relative h-32 w-32">
		<svg viewBox="0 0 100 100" class="h-full w-full -rotate-90" aria-hidden="true">
			<circle
				cx="50"
				cy="50"
				r={radius}
				fill="none"
				stroke="var(--t-bg-elevated)"
				stroke-width="8"
			/>
			{#if evaluation}
				<circle
					cx="50"
					cy="50"
					r={radius}
					fill="none"
					stroke={ringColor}
					stroke-width="8"
					stroke-linecap="round"
					stroke-dasharray={circ}
					stroke-dashoffset={dashOffset}
					style="transition: stroke-dashoffset 600ms cubic-bezier(0.32, 0.72, 0, 1);"
				/>
			{/if}
		</svg>
		<div class="absolute inset-0 flex flex-col items-center justify-center">
			{#if evaluation}
				<span
					class="font-display text-3xl font-extralight tracking-tight text-text-primary leading-none"
				>
					{Math.round(pct)}<span class="text-base text-text-muted">%</span>
				</span>
				{#if grade}
					<span class="mt-1 font-mono text-[11px] tracking-wide text-text-muted">
						{grade}
					</span>
				{/if}
			{:else}
				<span class="font-display text-2xl font-extralight text-text-muted">·</span>
			{/if}
		</div>
	</div>
	<span class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted capitalize">
		{category}
	</span>
</div>

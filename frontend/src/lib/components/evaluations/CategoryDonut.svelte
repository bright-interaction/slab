<script lang="ts">
	import type { Evaluation } from '$lib/api/types';
	import { asGrade, gradeTone, pctTone, toneCSSColor } from '$lib/evaluations/grade';

	let {
		category,
		evaluation,
		size = 'md'
	}: {
		category: string;
		evaluation: Evaluation | null;
		size?: 'sm' | 'md' | 'lg';
	} = $props();

	const grade = $derived(asGrade(evaluation?.grade));

	const pct = $derived.by<number>(() => {
		if (!evaluation || evaluation.max_score <= 0) return 0;
		const v = (evaluation.score / evaluation.max_score) * 100;
		if (Number.isNaN(v)) return 0;
		return Math.max(0, Math.min(100, v));
	});

	// Prefer the grade-derived tone (matches the badge); fall back to the
	// raw percentage so partially-loaded data still shows a sensible ring.
	const tone = $derived(grade ? gradeTone(grade) : pctTone(pct));
	const ringColor = $derived(evaluation ? toneCSSColor(tone) : toneCSSColor('muted'));

	const radius = 42;
	const circ = 2 * Math.PI * radius;
	const dashOffset = $derived(circ * (1 - pct / 100));

	const sizeClasses = $derived(
		size === 'lg' ? 'h-40 w-40' : size === 'sm' ? 'h-24 w-24' : 'h-32 w-32'
	);
	const pctTextClasses = $derived(
		size === 'lg' ? 'text-4xl' : size === 'sm' ? 'text-2xl' : 'text-3xl'
	);
</script>

<div class="flex flex-col items-center gap-2">
	<div class="relative {sizeClasses}">
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
					class="font-display {pctTextClasses} font-extralight tracking-tight text-text-primary leading-none"
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

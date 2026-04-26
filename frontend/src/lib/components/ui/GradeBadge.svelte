<script lang="ts">
	import type { Grade, Tone } from '$lib/evaluations/grade';
	import { gradeTone } from '$lib/evaluations/grade';

	type Size = 'sm' | 'md' | 'lg';

	let {
		grade,
		size = 'md',
		class: className = ''
	}: {
		grade: Grade;
		size?: Size;
		class?: string;
	} = $props();

	const tone: Tone = $derived(gradeTone(grade));

	const toneClasses: Record<Tone, string> = {
		success: 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 ring-1 ring-emerald-500/30',
		'success-soft':
			'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 ring-1 ring-emerald-500/20',
		warning: 'bg-amber-500/15 text-amber-600 dark:text-amber-400 ring-1 ring-amber-500/30',
		'warning-strong':
			'bg-orange-500/15 text-orange-600 dark:text-orange-400 ring-1 ring-orange-500/30',
		danger: 'bg-red-500/15 text-red-600 dark:text-red-400 ring-1 ring-red-500/30',
		muted: 'bg-bg-elevated text-text-muted ring-1 ring-border-light'
	};

	// 2-character grades ("A+", "B-") need a slightly smaller font so they
	// stay centred inside the same square as single-character "A".
	const isWide = $derived(grade.length === 2);

	const sizeClasses: Record<Size, string> = {
		sm: 'h-9 w-9',
		md: 'h-14 w-14',
		lg: 'h-20 w-20'
	};
	const fontClasses: Record<Size, { wide: string; narrow: string }> = {
		sm: { wide: 'text-sm', narrow: 'text-base' },
		md: { wide: 'text-xl', narrow: 'text-2xl' },
		lg: { wide: 'text-3xl', narrow: 'text-4xl' }
	};
</script>

<span
	class="inline-flex items-center justify-center rounded-2xl font-display font-extralight tracking-tight {toneClasses[
		tone
	]} {sizeClasses[size]} {isWide ? fontClasses[size].wide : fontClasses[size].narrow} {className}"
	aria-label={`Grade ${grade}`}
>
	{grade}
</span>

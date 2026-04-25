<script lang="ts">
	type Grade = 'A+' | 'A' | 'B+' | 'B' | 'C' | 'D' | 'F';
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

	const tone = $derived.by(() => {
		if (grade === 'A+' || grade === 'A') return 'success';
		if (grade === 'B+' || grade === 'B') return 'success-soft';
		if (grade === 'C') return 'warning';
		if (grade === 'D') return 'warning-strong';
		return 'danger';
	});

	const toneClasses: Record<string, string> = {
		success: 'bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 ring-1 ring-emerald-500/30',
		'success-soft':
			'bg-emerald-500/10 text-emerald-700 dark:text-emerald-300 ring-1 ring-emerald-500/20',
		warning: 'bg-amber-500/15 text-amber-600 dark:text-amber-400 ring-1 ring-amber-500/30',
		'warning-strong':
			'bg-orange-500/15 text-orange-600 dark:text-orange-400 ring-1 ring-orange-500/30',
		danger: 'bg-red-500/15 text-red-600 dark:text-red-400 ring-1 ring-red-500/30'
	};

	const sizeClasses: Record<Size, string> = {
		sm: 'h-9 w-9 text-base',
		md: 'h-14 w-14 text-2xl',
		lg: 'h-20 w-20 text-4xl'
	};
</script>

<span
	class="inline-flex items-center justify-center rounded-2xl font-display font-extralight tracking-tight {toneClasses[
		tone
	]} {sizeClasses[size]} {className}"
	aria-label={`Grade ${grade}`}
>
	{grade}
</span>

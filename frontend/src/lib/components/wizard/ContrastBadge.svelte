<script lang="ts">
	import { contrastRatio, passesAA, passesAAA } from '$lib/wcag';

	let {
		fg,
		bg,
		largeText = false,
		label
	}: {
		fg: string;
		bg: string;
		largeText?: boolean;
		label?: string;
	} = $props();

	const ratio = $derived(contrastRatio(fg, bg));
	const ratioText = $derived(`${ratio.toFixed(2)}:1`);
	const aa = $derived(passesAA(ratio, largeText));
	const aaa = $derived(passesAAA(ratio, largeText));

	const tone = $derived.by(() => {
		if (aa) return 'pass';
		if (ratio >= 3) return 'warn';
		return 'fail';
	});

	const toneClasses: Record<string, string> = {
		pass: 'border-success/30 bg-success/10 text-success',
		warn: 'border-warning/30 bg-warning/10 text-warning',
		fail: 'border-danger/30 bg-danger/10 text-danger'
	};
</script>

<span
	class="inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px] font-medium {toneClasses[
		tone
	]}"
>
	{#if label}
		<span class="text-text-secondary">{label}</span>
	{/if}
	<span>{ratioText}</span>
	<span class="text-[10px] uppercase tracking-wide opacity-80">
		{aaa ? 'AAA' : aa ? 'AA' : 'fail'}
	</span>
</span>

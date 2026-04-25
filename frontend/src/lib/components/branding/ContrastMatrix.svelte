<script lang="ts">
	import { contrastRatio, passesAA, suggestDarken } from '$lib/wcag';
	import ContrastBadge from './ContrastBadge.svelte';

	export type SlotKey =
		| 'primary_color'
		| 'secondary_color'
		| 'bg_color'
		| 'text_color';

	export interface ContrastColors {
		primary: string;
		secondary: string;
		bg: string;
		text: string;
	}

	let {
		colors,
		surfaceHex = '#ffffff',
		mutedHex = '#a3a3a3',
		onSuggest
	}: {
		colors: ContrastColors;
		surfaceHex?: string;
		mutedHex?: string;
		onSuggest: (slot: SlotKey, hex: string) => void;
	} = $props();

	interface Pair {
		label: string;
		fg: string;
		bg: string;
		fixSlot: SlotKey | null;
		fixOther: string | null;
	}

	const pairs = $derived<Pair[]>([
		{
			label: 'Text on bg',
			fg: colors.text,
			bg: colors.bg,
			fixSlot: 'text_color',
			fixOther: colors.bg
		},
		{
			label: 'Text on surface',
			fg: colors.text,
			bg: surfaceHex,
			fixSlot: 'text_color',
			fixOther: surfaceHex
		},
		{
			label: 'White on primary',
			fg: '#ffffff',
			bg: colors.primary,
			fixSlot: 'primary_color',
			fixOther: '#ffffff'
		},
		{
			label: 'White on secondary',
			fg: '#ffffff',
			bg: colors.secondary,
			fixSlot: 'secondary_color',
			fixOther: '#ffffff'
		},
		{
			label: 'Primary on bg',
			fg: colors.primary,
			bg: colors.bg,
			fixSlot: 'primary_color',
			fixOther: colors.bg
		},
		{
			label: 'Secondary on bg',
			fg: colors.secondary,
			bg: colors.bg,
			fixSlot: 'secondary_color',
			fixOther: colors.bg
		},
		{
			label: 'Muted on bg',
			fg: mutedHex,
			bg: colors.bg,
			fixSlot: null,
			fixOther: null
		}
	]);

	function fix(p: Pair): void {
		if (!p.fixSlot || !p.fixOther) return;
		const result = suggestDarken(p.fg, p.fixOther, 4.5);
		onSuggest(p.fixSlot, result.hex);
	}
</script>

<div class="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-4">
	{#each pairs as pair (pair.label)}
		{@const ratio = contrastRatio(pair.fg, pair.bg)}
		{@const ok = passesAA(ratio)}
		<div
			class="flex flex-col gap-2 rounded-lg border border-border-light bg-bg-surface p-3"
		>
			<div
				class="flex h-12 items-center justify-center rounded-md ring-1 ring-border-light"
				style:background-color={pair.bg}
			>
				<span
					class="text-[13px] font-medium"
					style:color={pair.fg}
				>
					{pair.label}
				</span>
			</div>
			<div class="flex items-center justify-between gap-2">
				<ContrastBadge fg={pair.fg} bg={pair.bg} />
				{#if !ok && pair.fixSlot}
					<button
						type="button"
						class="text-[11px] font-medium text-accent hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg rounded"
						onclick={() => fix(pair)}
					>
						Fix
					</button>
				{/if}
			</div>
		</div>
	{/each}
</div>

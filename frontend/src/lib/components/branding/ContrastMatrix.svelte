<script lang="ts">
	import { contrastRatio, passesAA, suggestDarken } from '$lib/wcag';
	import ContrastBadge from './ContrastBadge.svelte';
	import type { CssClass } from '$lib/api/types';

	export type SlotKey =
		| 'primary_color'
		| 'secondary_color'
		| 'bg_color'
		| 'text_color'
		| 'muted_color';

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
		cssClasses = [],
		onSuggest
	}: {
		colors: ContrastColors;
		surfaceHex?: string;
		mutedHex?: string;
		cssClasses?: CssClass[];
		onSuggest: (slot: SlotKey, hex: string) => void;
	} = $props();

	interface Pair {
		label: string;
		fg: string;
		bg: string;
		fixSlot: SlotKey | null;
		fixOther: string | null;
	}

	const builtInPairs = $derived<Pair[]>([
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
			fixSlot: 'muted_color',
			fixOther: colors.bg
		}
	]);

	const HEX_OR_RGB_RE = /(#[0-9a-fA-F]{3,8}|rgba?\([^)]+\))/;
	const COLOR_DECL_RE = /(?:^|[\s;{])color\s*:\s*([^;}\n]+)/i;
	const BG_DECL_RE = /(?:^|[\s;{])background(?:-color)?\s*:\s*([^;}\n]+)/i;

	function extractColor(decl: string): string | null {
		const m = decl.match(HEX_OR_RGB_RE);
		if (!m || !m[1]) return null;
		const v = m[1].trim();
		// Skip values we can't statically resolve (vars, gradients, color-mix).
		if (v.startsWith('var(') || v.includes('gradient') || v.includes('color-mix')) {
			return null;
		}
		return v;
	}

	const customPairs: Pair[] = $derived.by(() => {
		const out: Pair[] = [];
		for (const cls of cssClasses) {
			const fgMatch = cls.css.match(COLOR_DECL_RE);
			const bgMatch = cls.css.match(BG_DECL_RE);
			if (!fgMatch || !bgMatch || !fgMatch[1] || !bgMatch[1]) continue;
			const fg = extractColor(fgMatch[1]);
			const bg = extractColor(bgMatch[1]);
			if (!fg || !bg) continue;
			out.push({ label: cls.name, fg, bg, fixSlot: null, fixOther: null });
		}
		return out;
	});

	function fix(p: Pair): void {
		if (!p.fixSlot || !p.fixOther) return;
		const result = suggestDarken(p.fg, p.fixOther, 4.5);
		onSuggest(p.fixSlot, result.hex);
	}
</script>

{#snippet section(title: string, pairs: Pair[], emptyHint: string)}
	<div class="flex flex-col gap-3">
		<h3 class="text-[10.5px] font-mono uppercase tracking-[0.18em] text-text-secondary">
			{title}
		</h3>
		{#if pairs.length === 0}
			<p class="text-[11.5px] text-text-muted">{emptyHint}</p>
		{:else}
			<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
				{#each pairs as pair (pair.label)}
					{@const ratio = contrastRatio(pair.fg, pair.bg)}
					{@const ok = passesAA(ratio)}
					<div
						class="flex flex-col gap-2 rounded-lg border border-border-light bg-bg-surface p-3"
					>
						<div
							class="flex h-14 items-center justify-center rounded-md ring-1 ring-border-light"
							style:background-color={pair.bg}
						>
							<span class="truncate text-[14px] font-medium" style:color={pair.fg}>
								{pair.label}
							</span>
						</div>
						<div class="flex items-center justify-between gap-2">
							<ContrastBadge fg={pair.fg} bg={pair.bg} />
							{#if !ok && pair.fixSlot}
								<button
									type="button"
									class="rounded text-[11px] font-medium text-accent hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-bg"
									onclick={() => fix(pair)}
								>
									Fix
								</button>
							{:else if !ok}
								<span class="text-[11px] text-text-muted">Edit class to fix</span>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>
{/snippet}

<div class="flex flex-col gap-6">
	{@render section('Built-in palette', builtInPairs, '')}
	{@render section(
		'Custom CSS classes',
		customPairs,
		'No custom CSS classes with both color and background declarations yet. Pairs that use CSS vars, gradients, or color-mix() are skipped.'
	)}
</div>

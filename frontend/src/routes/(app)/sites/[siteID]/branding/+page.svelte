<script lang="ts">
	import * as sitesApi from '$lib/api/sites';
	import { ApiError } from '$lib/api/client';
	import Button from '$lib/components/ui/Button.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import ColorSlot from '$lib/components/branding/ColorSlot.svelte';
	import ContrastMatrix from '$lib/components/branding/ContrastMatrix.svelte';
	import type { SlotKey } from '$lib/components/branding/ContrastMatrix.svelte';
	import BrandingPreview from '$lib/components/branding/BrandingPreview.svelte';
	import { setSite } from '$lib/stores/currentSite.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import type { Site } from '$lib/api/types';

	let { data }: { data: { site: Site } } = $props();

	const DEFAULTS = {
		primary_color: '#0e7490',
		secondary_color: '#52525b',
		bg_color: '#fafaf9',
		text_color: '#18181b',
		font_heading: 'Inter',
		font_body: 'Inter'
	};

	const fontOptions = [
		{ value: 'Inter', label: 'Inter' },
		{ value: 'Geist', label: 'Geist' },
		{ value: 'Space Grotesk', label: 'Space Grotesk' },
		{ value: 'Playfair Display', label: 'Playfair Display' },
		{ value: 'Merriweather', label: 'Merriweather' },
		{ value: 'JetBrains Mono', label: 'JetBrains Mono' }
	];

	function initial(field: keyof typeof DEFAULTS): string {
		const v = data.site[field];
		return v && v.length > 0 ? v : DEFAULTS[field];
	}

	let primary = $state(initial('primary_color'));
	let secondary = $state(initial('secondary_color'));
	let bg = $state(initial('bg_color'));
	let text = $state(initial('text_color'));
	let fontHeading = $state(initial('font_heading'));
	let fontBody = $state(initial('font_body'));

	const initialState = {
		primary_color: initial('primary_color'),
		secondary_color: initial('secondary_color'),
		bg_color: initial('bg_color'),
		text_color: initial('text_color'),
		font_heading: initial('font_heading'),
		font_body: initial('font_body')
	};

	let saving = $state(false);

	const colors = $derived({ primary, secondary, bg, text });

	const dirty = $derived(
		primary !== initialState.primary_color ||
			secondary !== initialState.secondary_color ||
			bg !== initialState.bg_color ||
			text !== initialState.text_color ||
			fontHeading !== initialState.font_heading ||
			fontBody !== initialState.font_body
	);

	const HEX_RE = /^#[0-9a-fA-F]{6}$/;
	const valid = $derived(
		HEX_RE.test(primary) &&
			HEX_RE.test(secondary) &&
			HEX_RE.test(bg) &&
			HEX_RE.test(text)
	);

	function applySuggestion(slot: SlotKey, hex: string): void {
		const v = hex.toLowerCase();
		switch (slot) {
			case 'primary_color':
				primary = v;
				break;
			case 'secondary_color':
				secondary = v;
				break;
			case 'bg_color':
				bg = v;
				break;
			case 'text_color':
				text = v;
				break;
		}
	}

	function discard(): void {
		primary = initialState.primary_color;
		secondary = initialState.secondary_color;
		bg = initialState.bg_color;
		text = initialState.text_color;
		fontHeading = initialState.font_heading;
		fontBody = initialState.font_body;
	}

	async function save(): Promise<void> {
		if (saving || !dirty || !valid) return;
		saving = true;
		try {
			const updated = await sitesApi.update(data.site.id, {
				primary_color: primary,
				secondary_color: secondary,
				bg_color: bg,
				text_color: text,
				font_heading: fontHeading,
				font_body: fontBody
			});
			setSite(updated);
			initialState.primary_color = primary;
			initialState.secondary_color = secondary;
			initialState.bg_color = bg;
			initialState.text_color = text;
			initialState.font_heading = fontHeading;
			initialState.font_body = fontBody;
			toast.success('Branding saved.');
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Failed to save branding.';
			toast.error(msg);
		} finally {
			saving = false;
		}
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
			Branding
		</h1>
		<p class="text-[13px] text-text-secondary">
			Set the colors and fonts for this site. Changes apply to the next build.
		</p>
	</header>

	<div class="mt-8 grid grid-cols-1 gap-8 lg:grid-cols-[1fr_28rem]">
		<div class="flex flex-col gap-8">
			<section class="flex flex-col gap-5">
				<div class="flex items-baseline justify-between">
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Colors
					</h2>
				</div>
				<div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
					<ColorSlot
						label="Primary"
						helper="CTAs, links, focus rings."
						bind:value={primary}
						onReset={() => (primary = DEFAULTS.primary_color)}
					/>
					<ColorSlot
						label="Secondary"
						helper="Eyebrows, captions, footer text."
						bind:value={secondary}
						onReset={() => (secondary = DEFAULTS.secondary_color)}
					/>
					<ColorSlot
						label="Background"
						helper="Page surface behind everything."
						bind:value={bg}
						onReset={() => (bg = DEFAULTS.bg_color)}
					/>
					<ColorSlot
						label="Text"
						helper="Body copy, headings, default ink."
						bind:value={text}
						onReset={() => (text = DEFAULTS.text_color)}
					/>
				</div>
			</section>

			<section class="flex flex-col gap-3">
				<div class="flex items-baseline justify-between">
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Contrast matrix
					</h2>
					<span class="text-[11px] text-text-muted">WCAG AA target 4.5:1</span>
				</div>
				<ContrastMatrix {colors} onSuggest={applySuggestion} />
			</section>

			<section class="flex flex-col gap-5">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Typography
				</h2>
				<div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
					<div class="flex flex-col gap-1.5">
						<span class="text-[12px] font-medium text-text-primary">Heading font</span>
						<p class="text-[11px] text-text-muted">Used for H1 through H4.</p>
						<Select options={fontOptions} bind:value={fontHeading} />
					</div>
					<div class="flex flex-col gap-1.5">
						<span class="text-[12px] font-medium text-text-primary">Body font</span>
						<p class="text-[11px] text-text-muted">Body copy, navigation, buttons.</p>
						<Select options={fontOptions} bind:value={fontBody} />
					</div>
				</div>
			</section>

			<div
				class="sticky bottom-0 -mx-6 flex items-center justify-between gap-3 border-t border-border-light bg-bg-surface/85 px-6 py-3 backdrop-blur"
			>
				<span class="text-[12px] text-text-muted">
					{#if !valid}
						Some hex values are invalid.
					{:else if dirty}
						Unsaved changes.
					{:else}
						Up to date.
					{/if}
				</span>
				<div class="flex items-center gap-2">
					<Button variant="ghost" onclick={discard} disabled={!dirty || saving}>
						Discard
					</Button>
					<Button variant="primary" onclick={save} loading={saving} disabled={!dirty || !valid}>
						Save
					</Button>
				</div>
			</div>
		</div>

		<aside class="lg:sticky lg:top-24 lg:self-start">
			<BrandingPreview {primary} {secondary} {bg} {text} {fontHeading} {fontBody} />
		</aside>
	</div>
</div>

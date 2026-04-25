<script lang="ts">
	import Button from '$lib/components/ui/Button.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import { updateBranding, wizard } from '$lib/stores/wizard.svelte';
	import { contrastRatio, passesAA, suggestDarken } from '$lib/wcag';
	import ContrastBadge from '$lib/components/branding/ContrastBadge.svelte';

	const HEX_RE = /^#[0-9a-fA-F]{6}$/;

	type ColorKey = 'primary_color' | 'secondary_color' | 'bg_color' | 'text_color';

	interface ColorSlot {
		key: ColorKey;
		label: string;
	}

	const colorSlots: ColorSlot[] = [
		{ key: 'primary_color', label: 'Primary' },
		{ key: 'secondary_color', label: 'Secondary' },
		{ key: 'bg_color', label: 'Background' },
		{ key: 'text_color', label: 'Text' }
	];

	const fontOptions = [
		{ value: 'Inter', label: 'Inter' },
		{ value: 'Geist', label: 'Geist' },
		{ value: 'Space Grotesk', label: 'Space Grotesk' },
		{ value: 'Playfair Display', label: 'Playfair Display' },
		{ value: 'Merriweather', label: 'Merriweather' },
		{ value: 'JetBrains Mono', label: 'JetBrains Mono' }
	];

	let primary = $state(wizard.value.branding.primary_color);
	let secondary = $state(wizard.value.branding.secondary_color);
	let bg = $state(wizard.value.branding.bg_color);
	let text = $state(wizard.value.branding.text_color);
	let fontHeading = $state(wizard.value.branding.font_heading);
	let fontBody = $state(wizard.value.branding.font_body);

	function commit(): void {
		updateBranding({
			primary_color: normalize(primary),
			secondary_color: normalize(secondary),
			bg_color: normalize(bg),
			text_color: normalize(text),
			font_heading: fontHeading,
			font_body: fontBody
		});
	}

	function normalize(v: string): string {
		const trimmed = v.trim();
		if (HEX_RE.test(trimmed)) return trimmed.toLowerCase();
		return trimmed.startsWith('#') ? trimmed : `#${trimmed}`;
	}

	function setColor(key: ColorKey, value: string): void {
		const normalized = HEX_RE.test(value) ? value.toLowerCase() : value;
		switch (key) {
			case 'primary_color':
				primary = normalized;
				break;
			case 'secondary_color':
				secondary = normalized;
				break;
			case 'bg_color':
				bg = normalized;
				break;
			case 'text_color':
				text = normalized;
				break;
		}
		commit();
	}

	function getColor(key: ColorKey): string {
		switch (key) {
			case 'primary_color':
				return primary;
			case 'secondary_color':
				return secondary;
			case 'bg_color':
				return bg;
			case 'text_color':
				return text;
		}
	}

	$effect(() => {
		updateBranding({ font_heading: fontHeading, font_body: fontBody });
	});

	const textOnBgRatio = $derived(contrastRatio(text, bg));
	const whiteOnPrimaryRatio = $derived(contrastRatio('#ffffff', primary));

	const previewStyle = $derived(
		`--preview-primary:${primary};--preview-text:${text};--preview-bg:${bg};--preview-secondary:${secondary};--preview-font-heading:'${fontHeading}';--preview-font-body:'${fontBody}';`
	);

	function fixTextOnBg(): void {
		const result = suggestDarken(text, bg, 4.5);
		text = result.hex;
		commit();
	}

	function fixWhiteOnPrimary(): void {
		const result = suggestDarken(primary, '#ffffff', 4.5);
		primary = result.hex;
		commit();
	}
</script>

<div class="flex flex-col gap-6">
	<div class="flex flex-col gap-1">
		<h2 class="font-display text-2xl font-extralight tracking-tight text-text-primary">
			Branding
		</h2>
		<p class="text-[13px] text-text-secondary">
			Pick colors and fonts. We check WCAG contrast as you go.
		</p>
	</div>

	<div class="grid grid-cols-1 gap-6 lg:grid-cols-[1fr_1fr]">
		<div class="flex flex-col gap-5">
			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				{#each colorSlots as slot (slot.key)}
					<div class="flex flex-col gap-1.5">
						<label class="text-[12px] font-medium text-text-secondary" for="color-{slot.key}">
							{slot.label}
						</label>
						<div class="flex items-center gap-2">
							<input
								id="color-{slot.key}"
								type="color"
								class="h-9 w-12 cursor-pointer rounded-lg border border-border bg-bg-elevated p-0.5"
								value={getColor(slot.key)}
								oninput={(e) =>
									setColor(slot.key, (e.currentTarget as HTMLInputElement).value)}
							/>
							<input
								type="text"
								class="h-9 flex-1 rounded-lg border border-border bg-bg-elevated px-3 text-[13px] font-mono text-text-primary placeholder:text-text-muted focus:outline-none focus:border-accent focus:ring-1 focus:ring-accent"
								value={getColor(slot.key)}
								oninput={(e) =>
									setColor(slot.key, (e.currentTarget as HTMLInputElement).value)}
								spellcheck="false"
							/>
						</div>
					</div>
				{/each}
			</div>

			<div class="flex flex-col gap-2 rounded-xl border border-border-light bg-bg-surface p-4">
				<span class="text-[12px] font-medium text-text-secondary">Contrast checks</span>
				<div class="flex flex-wrap items-center gap-2">
					<ContrastBadge fg={text} {bg} label="Text on bg" />
					{#if !passesAA(textOnBgRatio)}
						<Button variant="secondary" size="sm" onclick={fixTextOnBg}>
							Darken to pass
						</Button>
					{/if}
				</div>
				<div class="flex flex-wrap items-center gap-2">
					<ContrastBadge fg="#ffffff" bg={primary} label="White on primary" />
					{#if !passesAA(whiteOnPrimaryRatio)}
						<Button variant="secondary" size="sm" onclick={fixWhiteOnPrimary}>
							Darken to pass
						</Button>
					{/if}
				</div>
				<div class="flex flex-wrap items-center gap-2">
					<ContrastBadge fg={secondary} {bg} label="Secondary on bg" />
				</div>
			</div>

			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div class="flex flex-col gap-1.5">
					<span class="text-[12px] font-medium text-text-secondary">Heading font</span>
					<Select options={fontOptions} bind:value={fontHeading} />
				</div>
				<div class="flex flex-col gap-1.5">
					<span class="text-[12px] font-medium text-text-secondary">Body font</span>
					<Select options={fontOptions} bind:value={fontBody} />
				</div>
			</div>
		</div>

		<div
			class="flex flex-col gap-4 overflow-hidden rounded-2xl border border-border-light p-8"
			style={previewStyle + 'background:var(--preview-bg);color:var(--preview-text);'}
		>
			<span
				class="text-[11px] font-medium uppercase tracking-wide"
				style="color:var(--preview-secondary);"
			>
				Preview
			</span>
			<h1
				style="font-family:var(--preview-font-heading);font-weight:200;letter-spacing:-0.025em;font-size:32px;line-height:1.1;color:var(--preview-text);"
			>
				Your headline goes here.
			</h1>
			<p
				style="font-family:var(--preview-font-body);font-size:14px;line-height:1.6;color:var(--preview-secondary);"
			>
				Subtitle copy renders in the body font. The button shows white on primary.
			</p>
			<div>
				<button
					type="button"
					class="inline-flex items-center justify-center rounded-lg px-4 py-2 text-[13px] font-medium transition-opacity hover:opacity-90"
					style="background:var(--preview-primary);color:#ffffff;font-family:var(--preview-font-body);"
				>
					Get started
				</button>
			</div>
		</div>
	</div>
</div>

<script lang="ts">
	import * as sitesApi from '$lib/api/sites';
	import * as fontsApi from '$lib/api/fonts';
	import * as designRefsApi from '$lib/api/designRefs';
	import * as cssClassesApi from '$lib/api/cssClasses';
	import { ApiError } from '$lib/api/client';
	import Button from '$lib/components/ui/Button.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import ColorSlot from '$lib/components/branding/ColorSlot.svelte';
	import ContrastMatrix from '$lib/components/branding/ContrastMatrix.svelte';
	import type { SlotKey } from '$lib/components/branding/ContrastMatrix.svelte';
	import { setSite } from '$lib/stores/currentSite.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { contrastRatio, passesAA } from '$lib/wcag';
	import { Trash2, RefreshCw, Github, Upload as UploadIcon, Eye } from 'lucide-svelte';
	import BrandingPreviewDialog from '$lib/components/branding/BrandingPreviewDialog.svelte';
	import type { CssClass, Site } from '$lib/api/types';

	let { data }: { data: { site: Site } } = $props();

	const DEFAULTS = {
		primary_color: '#0e7490',
		secondary_color: '#52525b',
		bg_color: '#fafaf9',
		text_color: '#18181b',
		surface_color: '#ffffff',
		border_color: '#e5e7eb',
		muted_color: '#6b7280',
		accent_color: '',
		on_primary_color: '#ffffff',
		font_heading: 'Inter',
		font_body: 'Inter'
	};

	// Curated set of self-hosted woff2-friendly fonts. Keep this list in
	// sync with the build pipeline's bundled fonts. Ten balanced choices
	// across geometric sans, humanist sans, mono, and serif.
	// Static option arrays for the Select component (consistent dropdown
	// chevron alignment vs native <select>).
	const weightOptions = [
		{ value: '100', label: '100 Thin' },
		{ value: '200', label: '200 ExtraLight' },
		{ value: '300', label: '300 Light' },
		{ value: '400', label: '400 Regular' },
		{ value: '500', label: '500 Medium' },
		{ value: '600', label: '600 SemiBold' },
		{ value: '700', label: '700 Bold' },
		{ value: '800', label: '800 ExtraBold' },
		{ value: '900', label: '900 Black' }
	];
	const styleOptions = [
		{ value: 'normal', label: 'Normal' },
		{ value: 'italic', label: 'Italic' }
	];
	const subsetOptions = [
		{ value: 'latin', label: 'Latin (default, ~80% smaller)' },
		{ value: 'latin-ext', label: 'Latin Extended' },
		{ value: 'cyrillic', label: 'Cyrillic' },
		{ value: 'greek', label: 'Greek' },
		{ value: 'vietnamese', label: 'Vietnamese' },
		{ value: 'arabic', label: 'Arabic' },
		{ value: 'hebrew', label: 'Hebrew' },
		{ value: 'all', label: 'All (no subsetting hint)' }
	];
	const refTypeOptions = [
		{ value: 'design-system', label: 'Design system' },
		{ value: 'component-library', label: 'Component library' },
		{ value: 'reference-site', label: 'Reference site' }
	];

	const builtInFonts = [
		{ value: 'Inter', label: 'Inter' },
		{ value: 'Geist', label: 'Geist' },
		{ value: 'Geist Mono', label: 'Geist Mono' },
		{ value: 'Space Grotesk', label: 'Space Grotesk' },
		{ value: 'Space Mono', label: 'Space Mono' },
		{ value: 'JetBrains Mono', label: 'JetBrains Mono' },
		{ value: 'Plus Jakarta Sans', label: 'Plus Jakarta Sans' },
		{ value: 'Manrope', label: 'Manrope' },
		{ value: 'IBM Plex Sans', label: 'IBM Plex Sans' },
		{ value: 'Playfair Display', label: 'Playfair Display' },
		{ value: 'Lora', label: 'Lora' },
		{ value: 'Source Serif 4', label: 'Source Serif 4' }
	];

	function initial(field: keyof typeof DEFAULTS): string {
		const v = data.site[field];
		return v && v.length > 0 ? v : DEFAULTS[field];
	}

	let primary = $state(initial('primary_color'));
	let secondary = $state(initial('secondary_color'));
	let bg = $state(initial('bg_color'));
	let text = $state(initial('text_color'));
	let surface = $state(initial('surface_color'));
	let border = $state(initial('border_color'));
	let muted = $state(initial('muted_color'));
	let accent = $state(initial('accent_color'));
	let onPrimary = $state(initial('on_primary_color'));
	let fontHeading = $state(initial('font_heading'));
	let fontBody = $state(initial('font_body'));

	const initialState = $state({
		primary_color: initial('primary_color'),
		secondary_color: initial('secondary_color'),
		bg_color: initial('bg_color'),
		text_color: initial('text_color'),
		surface_color: initial('surface_color'),
		border_color: initial('border_color'),
		muted_color: initial('muted_color'),
		accent_color: initial('accent_color'),
		on_primary_color: initial('on_primary_color'),
		font_heading: initial('font_heading'),
		font_body: initial('font_body')
	});

	let saving = $state(false);
	let previewOpen = $state(false);

	const colors = $derived({ primary, secondary, bg, text });

	// Mirror the 7 pairs ContrastMatrix renders so the header status strip
	// shows live AA pass count without coupling to the component.
	const aaPairs = $derived([
		contrastRatio(text, bg),
		contrastRatio(text, surface),
		contrastRatio('#ffffff', primary),
		contrastRatio('#ffffff', secondary),
		contrastRatio(primary, bg),
		contrastRatio(secondary, bg),
		contrastRatio(muted, bg)
	]);
	const aaPass = $derived(aaPairs.filter((r) => passesAA(r)).length);
	const aaTotal = $derived(aaPairs.length);

	// Custom fonts uploaded via /api/sites/{id}/fonts. Listed once on
	// mount and refreshed after every successful upload / delete.
	let customFonts = $state<fontsApi.SiteFont[]>([]);
	let customFontsLoading = $state(true);

	async function loadCustomFonts(): Promise<void> {
		customFontsLoading = true;
		try {
			const r = await fontsApi.list(data.site.id);
			customFonts = r.fonts;
		} catch {
			customFonts = [];
		} finally {
			customFontsLoading = false;
		}
	}

	let cssClasses = $state<CssClass[]>([]);

	async function loadCssClasses(): Promise<void> {
		try {
			cssClasses = await cssClassesApi.list(data.site.id);
		} catch {
			cssClasses = [];
		}
	}

	$effect(() => {
		void loadCustomFonts();
		void loadDesignRefs();
		void loadCssClasses();
	});

	const fontOptions = $derived(() => {
		const builtIns = builtInFonts.map((f) => ({ value: f.value, label: f.label }));
		const customFamilies = Array.from(
			new Set(customFonts.map((f) => f.family_name))
		).sort();
		const customOptions = customFamilies.map((fam) => ({
			value: fam,
			label: `${fam} (custom)`
		}));
		return [...builtIns, ...customOptions];
	});

	const dirty = $derived(
		primary !== initialState.primary_color ||
			secondary !== initialState.secondary_color ||
			bg !== initialState.bg_color ||
			text !== initialState.text_color ||
			surface !== initialState.surface_color ||
			border !== initialState.border_color ||
			muted !== initialState.muted_color ||
			accent !== initialState.accent_color ||
			onPrimary !== initialState.on_primary_color ||
			fontHeading !== initialState.font_heading ||
			fontBody !== initialState.font_body
	);

	const HEX_RE = /^#[0-9a-fA-F]{6}$/;
	const validOptional = (v: string) => v === '' || HEX_RE.test(v);
	const valid = $derived(
		HEX_RE.test(primary) &&
			HEX_RE.test(secondary) &&
			HEX_RE.test(bg) &&
			HEX_RE.test(text) &&
			HEX_RE.test(surface) &&
			HEX_RE.test(border) &&
			HEX_RE.test(muted) &&
			validOptional(accent) &&
			HEX_RE.test(onPrimary)
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
			case 'muted_color':
				muted = v;
				break;
		}
	}

	function discard(): void {
		primary = initialState.primary_color;
		secondary = initialState.secondary_color;
		bg = initialState.bg_color;
		text = initialState.text_color;
		surface = initialState.surface_color;
		border = initialState.border_color;
		muted = initialState.muted_color;
		accent = initialState.accent_color;
		onPrimary = initialState.on_primary_color;
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
				surface_color: surface,
				border_color: border,
				muted_color: muted,
				accent_color: accent,
				on_primary_color: onPrimary,
				font_heading: fontHeading,
				font_body: fontBody
			});
			setSite(updated);
			initialState.primary_color = primary;
			initialState.secondary_color = secondary;
			initialState.bg_color = bg;
			initialState.text_color = text;
			initialState.surface_color = surface;
			initialState.border_color = border;
			initialState.muted_color = muted;
			initialState.accent_color = accent;
			initialState.on_primary_color = onPrimary;
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

	// ---- Custom font upload ----
	let uploadFile = $state<File | null>(null);
	let uploadFamily = $state('');
	let uploadWeight = $state(400);
	let uploadStyle = $state<'normal' | 'italic'>('normal');
	let uploadSubset = $state<fontsApi.FontSubset>('latin');
	let uploading = $state(false);
	let uploadError = $state<string | null>(null);

	function onFilePick(e: Event): void {
		const target = e.currentTarget as HTMLInputElement;
		const f = target.files?.[0] ?? null;
		uploadFile = f;
		uploadError = null;
		if (f && !uploadFamily) {
			// Derive a family name from the filename (strip .woff2 + clean it up).
			const stem = f.name.replace(/\.woff2$/i, '').replace(/[._-]+/g, ' ').trim();
			uploadFamily = stem;
		}
		if (f && !f.name.toLowerCase().endsWith('.woff2')) {
			uploadError = 'Only .woff2 files are accepted (best compression, universal support).';
		}
	}

	async function uploadFont(): Promise<void> {
		if (uploading) return;
		if (!uploadFile) {
			uploadError = 'Pick a .woff2 file first.';
			return;
		}
		if (!uploadFamily.trim()) {
			uploadError = 'Family name required.';
			return;
		}
		uploading = true;
		uploadError = null;
		try {
			await fontsApi.upload(
				data.site.id,
				uploadFile,
				uploadFamily.trim(),
				uploadWeight,
				uploadStyle,
				uploadSubset
			);
			toast.success(
				`Uploaded ${uploadFamily.trim()} ${uploadWeight} ${uploadStyle} (${uploadSubset}).`
			);
			uploadFile = null;
			uploadFamily = '';
			uploadWeight = 400;
			uploadStyle = 'normal';
			uploadSubset = 'latin';
			await loadCustomFonts();
		} catch (err) {
			uploadError = err instanceof Error ? err.message : 'Upload failed.';
		} finally {
			uploading = false;
		}
	}

	async function removeFont(font: fontsApi.SiteFont): Promise<void> {
		try {
			await fontsApi.remove(data.site.id, font.id);
			toast.success(`Removed ${font.family_name} ${font.weight} ${font.style}.`);
			await loadCustomFonts();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to remove font.');
		}
	}

	function formatBytes(n: number): string {
		if (n < 1024) return `${n} B`;
		if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
		return `${(n / (1024 * 1024)).toFixed(2)} MB`;
	}

	// ---- Design references ----
	let designRefs = $state<designRefsApi.DesignReference[]>([]);
	let designRefsLoading = $state(true);

	let refURL = $state('');
	let refLabel = $state('');
	let refType = $state<designRefsApi.DesignReference['ref_type']>('design-system');
	let addingRef = $state(false);
	let refError = $state<string | null>(null);

	async function loadDesignRefs(): Promise<void> {
		designRefsLoading = true;
		try {
			const r = await designRefsApi.list(data.site.id);
			designRefs = r.references;
		} catch {
			designRefs = [];
		} finally {
			designRefsLoading = false;
		}
	}

	async function addDesignRef(): Promise<void> {
		if (addingRef || !refURL.trim()) return;
		addingRef = true;
		refError = null;
		try {
			await designRefsApi.create(data.site.id, refURL.trim(), refLabel.trim(), refType);
			toast.success('Design reference added. The AI agent now has its design vocabulary.');
			refURL = '';
			refLabel = '';
			refType = 'design-system';
			await loadDesignRefs();
		} catch (err) {
			refError = err instanceof ApiError ? err.message : 'Failed to add reference.';
		} finally {
			addingRef = false;
		}
	}

	async function refreshDesignRef(ref: designRefsApi.DesignReference): Promise<void> {
		try {
			await designRefsApi.refresh(data.site.id, ref.id);
			toast.success('Reference refreshed.');
			await loadDesignRefs();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to refresh.');
		}
	}

	async function removeDesignRef(ref: designRefsApi.DesignReference): Promise<void> {
		try {
			await designRefsApi.remove(data.site.id, ref.id);
			toast.success('Reference removed.');
			await loadDesignRefs();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to remove.');
		}
	}

	function fileCount(ref: designRefsApi.DesignReference): number {
		try {
			const j = JSON.parse(ref.fetched_json);
			return Array.isArray(j.files) ? j.files.length : 0;
		} catch {
			return 0;
		}
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex flex-wrap items-start justify-between gap-4">
		<div>
			<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
				Branding
			</h1>
			<p class="mt-1 text-[13px] text-text-secondary">
				Set the colors and fonts for this site. Changes apply to the next build.
			</p>
		</div>
		<div class="flex items-center gap-3">
			<div class="hidden flex-col items-end gap-0.5 sm:flex">
				<span
					class="text-[12px] {!valid
						? 'text-danger'
						: dirty
							? 'text-text-primary'
							: 'text-text-muted'}"
				>
					{#if !valid}
						Some hex values are invalid
					{:else if dirty}
						Unsaved changes
					{:else}
						Up to date
					{/if}
				</span>
				<span class="font-mono text-[10.5px] uppercase tracking-[0.12em] text-text-muted">
					{aaPass}/{aaTotal} AA · {customFonts.length}
					{customFonts.length === 1 ? 'font' : 'fonts'} · {designRefs.length}
					{designRefs.length === 1 ? 'ref' : 'refs'}
				</span>
			</div>
			<Button variant="ghost" onclick={() => (previewOpen = true)} title="Open live preview">
				<Eye class="mr-1 h-3.5 w-3.5" />
				Preview
			</Button>
			<Button variant="ghost" onclick={discard} disabled={!dirty || saving}>
				Discard
			</Button>
			<Button variant="primary" onclick={save} loading={saving} disabled={!dirty || !valid}>
				Save
			</Button>
		</div>
	</header>

	<div class="mt-8">
		<div class="flex flex-col gap-8">
			<section class="flex flex-col gap-6">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Colors
				</h2>

				<div class="flex flex-col gap-3">
					<h3 class="text-[10.5px] font-mono uppercase tracking-[0.18em] text-text-secondary">
						Brand
					</h3>
					<div class="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
						<ColorSlot
							label="Primary"
							helper="CTAs, focus rings."
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
							label="Accent"
							helper="Links, focus accents. Empty = use primary."
							bind:value={accent}
							onReset={() => (accent = DEFAULTS.accent_color)}
						/>
					</div>
				</div>

				<div class="flex flex-col gap-3">
					<h3 class="text-[10.5px] font-mono uppercase tracking-[0.18em] text-text-secondary">
						Surface
					</h3>
					<div class="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
						<ColorSlot
							label="Background"
							helper="Page surface behind everything."
							bind:value={bg}
							onReset={() => (bg = DEFAULTS.bg_color)}
						/>
						<ColorSlot
							label="Surface"
							helper="Card panels, elevated sections."
							bind:value={surface}
							onReset={() => (surface = DEFAULTS.surface_color)}
						/>
						<ColorSlot
							label="Border"
							helper="Hairlines, card outlines, dividers."
							bind:value={border}
							onReset={() => (border = DEFAULTS.border_color)}
						/>
					</div>
				</div>

				<div class="flex flex-col gap-3">
					<h3 class="text-[10.5px] font-mono uppercase tracking-[0.18em] text-text-secondary">
						Text
					</h3>
					<div class="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
						<ColorSlot
							label="Text"
							helper="Body copy, headings, default ink."
							bind:value={text}
							onReset={() => (text = DEFAULTS.text_color)}
						/>
						<ColorSlot
							label="Muted text"
							helper="Captions, hints, de-emphasised copy."
							bind:value={muted}
							onReset={() => (muted = DEFAULTS.muted_color)}
						/>
						<ColorSlot
							label="On primary"
							helper="Text on primary buttons. Use black for light primaries."
							bind:value={onPrimary}
							onReset={() => (onPrimary = DEFAULTS.on_primary_color)}
						/>
					</div>
				</div>
			</section>

			<section class="flex flex-col gap-3">
				<div class="flex items-baseline justify-between">
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Contrast matrix
					</h2>
					<span class="text-[11px] text-text-muted">WCAG AA target 4.5:1</span>
				</div>
				<ContrastMatrix
					{colors}
					surfaceHex={surface}
					mutedHex={muted}
					{cssClasses}
					onSuggest={applySuggestion}
				/>
			</section>

			<section class="flex flex-col gap-5">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Typography
				</h2>
				<div class="grid grid-cols-1 gap-5 sm:grid-cols-2">
					<div class="flex flex-col gap-1.5">
						<span class="text-[12px] font-medium text-text-primary">Heading font</span>
						<p class="text-[11px] text-text-muted">Used for H1 through H4.</p>
						<Select options={fontOptions()} bind:value={fontHeading} />
					</div>
					<div class="flex flex-col gap-1.5">
						<span class="text-[12px] font-medium text-text-primary">Body font</span>
						<p class="text-[11px] text-text-muted">Body copy, navigation, buttons.</p>
						<Select options={fontOptions()} bind:value={fontBody} />
					</div>
				</div>
			</section>

			<section class="flex flex-col gap-3">
				<div class="flex items-baseline justify-between">
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Custom fonts
					</h2>
					<span class="text-[11px] text-text-muted">.woff2 only</span>
				</div>

				<Card padding="md">
					<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
						<div class="sm:col-span-2">
							<label
								class="flex cursor-pointer items-center justify-between gap-3 rounded-lg border border-dashed border-border-light bg-bg-elevated p-3 hover:bg-bg-hover"
							>
								<div class="flex items-center gap-2 text-[13px] text-text-secondary">
									<UploadIcon size={16} strokeWidth={1.75} />
									{uploadFile ? uploadFile.name : 'Pick a .woff2 file...'}
								</div>
								<span class="text-[11px] text-text-muted">
									{uploadFile ? formatBytes(uploadFile.size) : 'click to choose'}
								</span>
								<input
									type="file"
									accept=".woff2"
									class="hidden"
									onchange={onFilePick}
								/>
							</label>
						</div>
						<Input label="Family name" placeholder="Acme Sans" bind:value={uploadFamily} />
						<div class="grid grid-cols-2 gap-3">
							<div class="flex flex-col gap-1.5">
								<span class="text-[12px] font-medium text-text-primary">Weight</span>
								<Select
									options={weightOptions}
									value={String(uploadWeight)}
									onchange={(v) => (uploadWeight = Number(v))}
								/>
							</div>
							<div class="flex flex-col gap-1.5">
								<span class="text-[12px] font-medium text-text-primary">Style</span>
								<Select
									options={styleOptions}
									value={uploadStyle}
									onchange={(v) => (uploadStyle = v as 'normal' | 'italic')}
								/>
							</div>
						</div>
						<div class="flex flex-col gap-1.5">
							<span class="text-[12px] font-medium text-text-primary">Unicode range</span>
							<Select
								options={subsetOptions}
								value={uploadSubset}
								onchange={(v) => (uploadSubset = v as fontsApi.FontSubset)}
							/>
							<p class="text-[11px] text-text-muted">
								Browser only fetches the font when the page contains characters in
								this range. Saves ~80% of font weight on a Latin-only EU/UK site.
							</p>
						</div>
					</div>
					{#if uploadError}
						<p class="mt-3 text-[12px] text-danger">{uploadError}</p>
					{/if}
					<div class="mt-3 flex flex-wrap items-center justify-between gap-2">
						<p class="text-[11px] text-text-muted">
							Self-hosted from /atomicsite-fonts. No external CDN.
						</p>
						<Button
							variant="primary"
							onclick={uploadFont}
							loading={uploading}
							disabled={uploading || !uploadFile}
							class="whitespace-nowrap"
						>
							Upload font
						</Button>
					</div>
				</Card>

				{#if customFontsLoading}
					<p class="text-[12px] text-text-muted">Loading...</p>
				{:else if customFonts.length > 0}
					<Card padding="none">
						<ul class="divide-y divide-border-light">
							{#each customFonts as f (f.id)}
								<li class="flex items-center gap-4 px-4 py-3">
									<div class="min-w-0 flex-1">
										<p
											class="truncate text-[13px] text-text-primary"
											style="font-family: '{f.family_name}', system-ui, sans-serif;"
										>
											{f.family_name}
										</p>
										<p class="mt-0.5 text-[11px] text-text-muted">
											{f.weight}
											{f.style}
											{#if f.original_name}
												· <span class="font-mono">{f.original_name}</span>
											{/if}
											· {formatBytes(f.file_size)}
										</p>
									</div>
									<button
										type="button"
										aria-label="Remove font"
										class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-text-muted hover:bg-danger/10 hover:text-danger"
										onclick={() => removeFont(f)}
									>
										<Trash2 size={14} strokeWidth={1.75} />
									</button>
								</li>
							{/each}
						</ul>
					</Card>
				{/if}
			</section>

			<section class="flex flex-col gap-3">
				<div class="flex items-baseline justify-between">
					<div>
						<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
							Design references
						</h2>
						<p class="mt-1 text-[12px] text-text-muted">
							Public GitHub repos that the AI agent reads as design vocabulary. Atomicsite
							pre-fetches package.json, README, tailwind config, global stylesheet, and a few
							component files. Read-only pattern reference, never code copy.
						</p>
					</div>
					<Github size={16} strokeWidth={1.75} class="text-text-muted" />
				</div>

				<Card padding="md">
					<div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
						<div class="sm:col-span-2">
							<Input
								label="GitHub repo URL"
								placeholder="https://github.com/owner/repo"
								bind:value={refURL}
							/>
						</div>
						<Input label="Label (optional)" placeholder="e.g. Taste UI" bind:value={refLabel} />
						<div class="flex flex-col gap-1.5 sm:col-span-2">
							<span class="text-[12px] font-medium text-text-primary">Type</span>
							<Select
								options={refTypeOptions}
								value={refType}
								onchange={(v) =>
									(refType = v as designRefsApi.DesignReference['ref_type'])}
							/>
						</div>
						<div class="flex items-end justify-end">
							<Button
								variant="primary"
								onclick={addDesignRef}
								loading={addingRef}
								disabled={addingRef || !refURL.trim()}
								class="whitespace-nowrap"
							>
								Add reference
							</Button>
						</div>
					</div>
					{#if refError}
						<p class="mt-3 text-[12px] text-danger">{refError}</p>
					{/if}
				</Card>

				{#if designRefsLoading}
					<p class="text-[12px] text-text-muted">Loading...</p>
				{:else if designRefs.length > 0}
					<Card padding="none">
						<ul class="divide-y divide-border-light">
							{#each designRefs as r (r.id)}
								<li class="flex items-center gap-3 px-4 py-3">
									<div class="min-w-0 flex-1">
										<div class="flex flex-wrap items-center gap-2">
											<a
												href={r.url}
												target="_blank"
												rel="noopener noreferrer"
												class="truncate text-[13px] font-medium text-text-primary hover:underline"
											>
												{r.label || r.url.replace('https://github.com/', '')}
											</a>
											<Badge variant="default">{r.ref_type}</Badge>
											<span class="text-[11px] text-text-muted">
												{fileCount(r)} file{fileCount(r) === 1 ? '' : 's'} fetched
											</span>
										</div>
										<p class="mt-0.5 truncate font-mono text-[11px] text-text-muted">
											{r.url}
										</p>
									</div>
									<button
										type="button"
										aria-label="Refresh reference"
										class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-text-muted hover:bg-bg-hover hover:text-text-primary"
										onclick={() => refreshDesignRef(r)}
									>
										<RefreshCw size={14} strokeWidth={1.75} />
									</button>
									<button
										type="button"
										aria-label="Remove reference"
										class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md text-text-muted hover:bg-danger/10 hover:text-danger"
										onclick={() => removeDesignRef(r)}
									>
										<Trash2 size={14} strokeWidth={1.75} />
									</button>
								</li>
							{/each}
						</ul>
					</Card>
				{/if}
			</section>
		</div>

	</div>
</div>

<BrandingPreviewDialog
	bind:open={previewOpen}
	{primary}
	{secondary}
	{bg}
	{text}
	{surface}
	{border}
	{muted}
	{accent}
	{onPrimary}
	{fontHeading}
	{fontBody}
/>

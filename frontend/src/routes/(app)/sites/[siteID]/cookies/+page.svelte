<script lang="ts">
	import * as settingsApi from '$lib/api/settings';
	import { ApiError } from '$lib/api/client';
	import Input from '$lib/components/ui/Input.svelte';
	import Switch from '$lib/components/ui/Switch.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import Textarea from '$lib/components/ui/Textarea.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { categoryMap } from '$lib/settings/nginxPreview';
	import type { Site } from '$lib/api/types';
	import type { CookieDeclaration, CookieTranslation } from '$lib/api/settings';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);

	function toBool(v: string | undefined, fallback = false): boolean {
		if (v === undefined || v === '') return fallback;
		const s = v.toLowerCase();
		return s === '1' || s === 'true' || s === 'yes' || s === 'on';
	}

	function clampInt(v: string | undefined, fallback: number, min: number, max: number): number {
		if (v === undefined || v === '') return fallback;
		const n = Number.parseInt(v, 10);
		if (!Number.isFinite(n)) return fallback;
		if (n < min || n > max) return fallback;
		return n;
	}

	type CategoryID = 'necessary' | 'analytics' | 'marketing' | 'preferences';
	const CATEGORY_ORDER: CategoryID[] = ['necessary', 'analytics', 'marketing', 'preferences'];
	const CATEGORY_LABELS: Record<CategoryID, string> = {
		necessary: 'Necessary',
		analytics: 'Analytics',
		marketing: 'Marketing',
		preferences: 'Preferences'
	};

	let loading = $state(true);
	let saving = $state(false);

	let enabled = $state(false);
	let position = $state<'bottom' | 'top' | 'center'>('bottom');
	let title = $state('');
	let description = $state('');
	let acceptLabel = $state('');
	let rejectLabel = $state('');
	let customizeLabel = $state('');
	let catAnalytics = $state(true);
	let catMarketing = $state(true);
	let catPreferences = $state(true);

	// Compliance & UX (added 2026-05-01).
	let cookieExpiryDays = $state(365);
	let cookieRevision = $state(0);
	let cookieTheme = $state<'light' | 'dark' | 'auto'>('light');
	let floatingTrigger = $state<'left' | 'right' | 'off'>('left');
	let languageSelector = $state(false);
	// 14 ISO codes the embedded CookieProof bundle ships translations for.
	// Order them with the site's own language first (resolved at load time)
	// so the form puts the most-relevant choice on the left.
	const SUPPORTED_LANGUAGES: { code: string; label: string }[] = [
		{ code: 'en', label: 'English' },
		{ code: 'sv', label: 'Svenska' },
		{ code: 'da', label: 'Dansk' },
		{ code: 'de', label: 'Deutsch' },
		{ code: 'es', label: 'Español' },
		{ code: 'fi', label: 'Suomi' },
		{ code: 'fr', label: 'Français' },
		{ code: 'it', label: 'Italiano' },
		{ code: 'ja', label: '日本語' },
		{ code: 'nl', label: 'Nederlands' },
		{ code: 'no', label: 'Norsk' },
		{ code: 'pl', label: 'Polski' },
		{ code: 'pt', label: 'Português' }
	];
	// Selected ISO codes (persisted as comma-separated in
	// analytics.cookie_languages). Empty list + languageSelector=true
	// means "all built-in languages"; empty list + languageSelector=false
	// hides the dropdown entirely.
	let selectedLanguages = $state<string[]>([]);
	let ccpaEnabled = $state(false);
	let ccpaUrl = $state('');

	let userDeclarations = $state<CookieDeclaration[]>([]);
	let presets = $state<CookieDeclaration[]>([]);

	// Per-language string overrides. Mirrors CookieProof's customTranslations
	// shape: { sv: { title, description, accept, reject, customize }, ... }.
	// Empty fields fall back to the widget's built-in translation for that
	// locale, so the operator only fills what they want to override.
	// Persisted as JSON in analytics.cookie_translations.
	let translations = $state<Record<string, CookieTranslation>>({});
	let editingLang = $state<string>('');

	function ensureLang(code: string): void {
		if (!translations[code]) {
			translations = { ...translations, [code]: {} };
		}
		editingLang = code;
	}

	function clearLang(code: string): void {
		const next = { ...translations };
		delete next[code];
		translations = next;
		if (editingLang === code) editingLang = '';
	}

	function tField(code: string, key: keyof CookieTranslation): string {
		return translations[code]?.[key] ?? '';
	}

	function setTField(code: string, key: keyof CookieTranslation, value: string): void {
		const cur = translations[code] ?? {};
		const nextEntry: CookieTranslation = { ...cur, [key]: value };
		// Strip empty-string keys so the JSON stays small and the widget
		// merge logic falls back cleanly to built-ins.
		for (const k of Object.keys(nextEntry) as (keyof CookieTranslation)[]) {
			if (!nextEntry[k]) delete nextEntry[k];
		}
		translations = { ...translations, [code]: nextEntry };
	}

	function serializeTranslations(t: Record<string, CookieTranslation>): string {
		// Drop empty per-lang entries so the JSON stays tight.
		const out: Record<string, CookieTranslation> = {};
		for (const [code, entry] of Object.entries(t)) {
			if (entry && Object.values(entry).some((v) => typeof v === 'string' && v.length > 0)) {
				out[code] = entry;
			}
		}
		return JSON.stringify(out);
	}

	function parseTranslations(raw: string): Record<string, CookieTranslation> {
		const trimmed = (raw || '').trim();
		if (!trimmed) return {};
		try {
			const parsed = JSON.parse(trimmed);
			if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return {};
			const out: Record<string, CookieTranslation> = {};
			for (const [code, entry] of Object.entries(parsed)) {
				if (entry && typeof entry === 'object' && !Array.isArray(entry)) {
					const e = entry as Record<string, unknown>;
					const t: CookieTranslation = {};
					for (const k of ['title', 'description', 'accept', 'reject', 'customize'] as const) {
						const v = e[k];
						if (typeof v === 'string' && v.length > 0) t[k] = v;
					}
					if (Object.keys(t).length > 0) out[code] = t;
				}
			}
			return out;
		} catch {
			return {};
		}
	}

	type State = {
		enabled: boolean;
		position: 'bottom' | 'top' | 'center';
		title: string;
		description: string;
		acceptLabel: string;
		rejectLabel: string;
		customizeLabel: string;
		catAnalytics: boolean;
		catMarketing: boolean;
		catPreferences: boolean;
		declarationsJSON: string;
		cookieExpiryDays: number;
		cookieRevision: number;
		cookieTheme: 'light' | 'dark' | 'auto';
		floatingTrigger: 'left' | 'right' | 'off';
		languageSelector: boolean;
		languages: string;
		translationsJSON: string;
		ccpaEnabled: boolean;
		ccpaUrl: string;
	};

	let initial: State = $state({
		enabled: false,
		position: 'bottom',
		title: '',
		description: '',
		acceptLabel: '',
		rejectLabel: '',
		customizeLabel: '',
		catAnalytics: true,
		catMarketing: true,
		catPreferences: true,
		declarationsJSON: '[]',
		cookieExpiryDays: 365,
		cookieRevision: 0,
		cookieTheme: 'light',
		floatingTrigger: 'left',
		languageSelector: false,
		languages: '',
		translationsJSON: '{}',
		ccpaEnabled: false,
		ccpaUrl: ''
	});

	function serializeDeclarations(rows: CookieDeclaration[]): string {
		const cleaned = rows
			.filter((r) => r.name.trim() !== '')
			.map((r) => ({
				category: r.category,
				name: r.name.trim(),
				provider: (r.provider ?? '').trim() || undefined,
				purpose: (r.purpose ?? '').trim() || undefined,
				expiry: (r.expiry ?? '').trim() || undefined
			}));
		return cleaned.length === 0 ? '' : JSON.stringify(cleaned);
	}

	function parseDeclarations(raw: string): CookieDeclaration[] {
		const v = (raw ?? '').trim();
		if (v === '') return [];
		try {
			const parsed = JSON.parse(v);
			if (!Array.isArray(parsed)) return [];
			return parsed.filter(
				(r) => r && typeof r.name === 'string' && CATEGORY_ORDER.includes(r.category)
			);
		} catch {
			return [];
		}
	}

	async function load() {
		loading = true;
		try {
			const [rows, presetsResp] = await Promise.all([
				settingsApi.listByCategory(siteID, 'analytics'),
				settingsApi.getCookiePresets(siteID).catch(() => [] as CookieDeclaration[])
			]);
			const m = categoryMap(rows);

			enabled = toBool(m.cookieproof_enabled);
			const pos = (m.cookie_banner_position || 'bottom').toLowerCase();
			position = pos === 'top' || pos === 'center' ? pos : 'bottom';
			title = m.cookie_banner_title || '';
			description = m.cookie_banner_description || '';
			acceptLabel = m.cookie_banner_accept || '';
			rejectLabel = m.cookie_banner_reject || '';
			customizeLabel = m.cookie_banner_customize || '';
			catAnalytics = toBool(m.cookie_cat_analytics, true);
			catMarketing = toBool(m.cookie_cat_marketing, true);
			catPreferences = toBool(m.cookie_cat_preferences, true);

			cookieExpiryDays = clampInt(m.cookie_expiry_days, 365, 0, 365);
			cookieRevision = clampInt(m.cookie_revision, 0, 0, 9999);
			const t = (m.cookie_theme || 'light').toLowerCase();
			cookieTheme = t === 'dark' || t === 'auto' ? (t as 'dark' | 'auto') : 'light';
			const ft = (m.cookie_floating_trigger || 'left').toLowerCase();
			floatingTrigger = ft === 'right' || ft === 'off' ? (ft as 'right' | 'off') : 'left';
			languageSelector = toBool(m.cookie_language_selector, false);
			selectedLanguages = (m.cookie_languages || '')
				.split(',')
				.map((s) => s.trim().toLowerCase())
				.filter((s) => s && SUPPORTED_LANGUAGES.some((l) => l.code === s));
			ccpaEnabled = toBool(m.ccpa_enabled, false);
			ccpaUrl = m.ccpa_url || '';

			userDeclarations = parseDeclarations(m.cookie_declarations || '');
			presets = presetsResp;
			translations = parseTranslations(m.cookie_translations || '');
			editingLang = '';

			initial = {
				enabled,
				position,
				title,
				description,
				acceptLabel,
				rejectLabel,
				customizeLabel,
				catAnalytics,
				catMarketing,
				catPreferences,
				declarationsJSON: serializeDeclarations(userDeclarations),
				cookieExpiryDays,
				cookieRevision,
				cookieTheme,
				floatingTrigger,
				languageSelector,
				languages: selectedLanguages.join(','),
				translationsJSON: serializeTranslations(translations),
				ccpaEnabled,
				ccpaUrl
			};
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to load cookie settings.');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	const declarationsJSONNow = $derived(serializeDeclarations(userDeclarations));
	const translationsJSONNow = $derived(serializeTranslations(translations));

	const dirty = $derived(
		enabled !== initial.enabled ||
			position !== initial.position ||
			title !== initial.title ||
			description !== initial.description ||
			acceptLabel !== initial.acceptLabel ||
			rejectLabel !== initial.rejectLabel ||
			customizeLabel !== initial.customizeLabel ||
			catAnalytics !== initial.catAnalytics ||
			catMarketing !== initial.catMarketing ||
			catPreferences !== initial.catPreferences ||
			declarationsJSONNow !== initial.declarationsJSON ||
			cookieExpiryDays !== initial.cookieExpiryDays ||
			cookieRevision !== initial.cookieRevision ||
			cookieTheme !== initial.cookieTheme ||
			floatingTrigger !== initial.floatingTrigger ||
			languageSelector !== initial.languageSelector ||
			selectedLanguages.join(',') !== initial.languages ||
			translationsJSONNow !== initial.translationsJSON ||
			ccpaEnabled !== initial.ccpaEnabled ||
			ccpaUrl !== initial.ccpaUrl
	);

	function discard() {
		enabled = initial.enabled;
		position = initial.position;
		title = initial.title;
		description = initial.description;
		acceptLabel = initial.acceptLabel;
		rejectLabel = initial.rejectLabel;
		customizeLabel = initial.customizeLabel;
		catAnalytics = initial.catAnalytics;
		catMarketing = initial.catMarketing;
		catPreferences = initial.catPreferences;
		userDeclarations = parseDeclarations(initial.declarationsJSON);
		cookieExpiryDays = initial.cookieExpiryDays;
		cookieRevision = initial.cookieRevision;
		cookieTheme = initial.cookieTheme;
		floatingTrigger = initial.floatingTrigger;
		languageSelector = initial.languageSelector;
		selectedLanguages = initial.languages
			? initial.languages.split(',').map((s) => s.trim().toLowerCase()).filter(Boolean)
			: [];
		translations = parseTranslations(initial.translationsJSON);
		editingLang = '';
		ccpaEnabled = initial.ccpaEnabled;
		ccpaUrl = initial.ccpaUrl;
	}

	function b(v: boolean): string {
		return v ? '1' : '0';
	}

	function addDeclaration(category: CategoryID) {
		userDeclarations = [
			...userDeclarations,
			{ category, name: '', provider: '', purpose: '', expiry: '' }
		];
	}

	function removeDeclaration(idx: number) {
		userDeclarations = userDeclarations.filter((_, i) => i !== idx);
	}

	function overridePreset(p: CookieDeclaration) {
		// Copy the preset into the user list so the strings become editable.
		// Same (category, name) collision: backend merge gives the user row
		// precedence and the preset row drops out of the rendered table.
		userDeclarations = [...userDeclarations, { ...p }];
	}

	function userRowsFor(category: CategoryID): { row: CookieDeclaration; idx: number }[] {
		return userDeclarations
			.map((row, idx) => ({ row, idx }))
			.filter(({ row }) => row.category === category);
	}

	function presetRowsFor(category: CategoryID): CookieDeclaration[] {
		return presets.filter((p) => p.category === category);
	}

	function isOverridden(p: CookieDeclaration): boolean {
		return userDeclarations.some(
			(u) => u.category === p.category && u.name.trim() === p.name.trim()
		);
	}

	async function save() {
		if (saving || !dirty) return;
		saving = true;
		try {
			const items: settingsApi.SettingUpsertInput[] = [
				{ category: 'analytics', key: 'cookieproof_enabled', value: b(enabled) },
				{ category: 'analytics', key: 'cookie_banner_position', value: position },
				{ category: 'analytics', key: 'cookie_banner_title', value: title },
				{ category: 'analytics', key: 'cookie_banner_description', value: description },
				{ category: 'analytics', key: 'cookie_expiry_days', value: String(cookieExpiryDays) },
				{ category: 'analytics', key: 'cookie_revision', value: String(cookieRevision) },
				{ category: 'analytics', key: 'cookie_theme', value: cookieTheme },
				{ category: 'analytics', key: 'cookie_floating_trigger', value: floatingTrigger },
				{ category: 'analytics', key: 'cookie_language_selector', value: b(languageSelector) },
				{ category: 'analytics', key: 'cookie_languages', value: selectedLanguages.join(',') },
				{ category: 'analytics', key: 'ccpa_enabled', value: b(ccpaEnabled) },
				{ category: 'analytics', key: 'ccpa_url', value: ccpaUrl },
				{ category: 'analytics', key: 'cookie_banner_accept', value: acceptLabel },
				{ category: 'analytics', key: 'cookie_banner_reject', value: rejectLabel },
				{ category: 'analytics', key: 'cookie_banner_customize', value: customizeLabel },
				{ category: 'analytics', key: 'cookie_cat_analytics', value: b(catAnalytics) },
				{ category: 'analytics', key: 'cookie_cat_marketing', value: b(catMarketing) },
				{ category: 'analytics', key: 'cookie_cat_preferences', value: b(catPreferences) },
				{ category: 'analytics', key: 'cookie_declarations', value: declarationsJSONNow },
				{ category: 'analytics', key: 'cookie_translations', value: translationsJSONNow }
			];
			await settingsApi.bulkUpsert(siteID, items);
			initial = {
				enabled,
				position,
				title,
				description,
				acceptLabel,
				rejectLabel,
				customizeLabel,
				catAnalytics,
				catMarketing,
				catPreferences,
				declarationsJSON: declarationsJSONNow,
				cookieExpiryDays,
				cookieRevision,
				cookieTheme,
				floatingTrigger,
				languageSelector,
				languages: selectedLanguages.join(','),
				translationsJSON: translationsJSONNow,
				ccpaEnabled,
				ccpaUrl
			};
			toast.success('Cookie banner settings saved. Rebuild to publish.');
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Failed to save settings.';
			toast.error(msg);
		} finally {
			saving = false;
		}
	}

	// Live preview iframe URL. Pulls in unsaved form state via query
	// params so editors see the real CookieProof widget update as they
	// type. Reload is debounced via the keyed reactive expression.
	let previewBust = $state(0);
	const previewSrc = $derived.by(() => {
		void previewBust; // bust dep
		const params = new URLSearchParams();
		const set = (k: string, v: string | undefined | null) => {
			if (v !== undefined && v !== null && v !== '') params.set(k, v);
		};
		set('title', title);
		set('description', description);
		set('accept_label', acceptLabel);
		set('reject_label', rejectLabel);
		set('customize_label', customizeLabel);
		set('language_selector', languageSelector ? '1' : '0');
		set('languages', selectedLanguages.join(','));
		set('translations', translationsJSONNow);
		set('theme', cookieTheme);
		set('floating_trigger', floatingTrigger);
		set('position', position);
		set('ccpa_enabled', ccpaEnabled ? '1' : '0');
		set('ccpa_url', ccpaUrl);
		// Branding pulled live from the site row (the Branding page edits
		// data.site directly via the Save button there; we forward the
		// current values regardless so the preview always matches).
		set('primary_color', data.site.primary_color);
		set('secondary_color', data.site.secondary_color);
		set('on_primary_color', data.site.on_primary_color);
		set('text_color', data.site.text_color);
		set('bg_color', data.site.bg_color);
		set('surface_color', data.site.surface_color);
		set('muted_color', data.site.muted_color);
		set('border_color', data.site.border_color);
		set('font_body', data.site.font_body);
		return `/api/sites/${siteID}/cookies/preview?${params.toString()}&_=${previewBust}`;
	});

	// Debounce preview iframe reload by 350ms after the last edit.
	let previewTimer: ReturnType<typeof setTimeout> | null = null;
	function bumpPreview() {
		if (previewTimer) clearTimeout(previewTimer);
		previewTimer = setTimeout(() => {
			previewBust++;
			previewTimer = null;
		}, 350);
	}
	$effect(() => {
		// Track the fields that should trigger a reload.
		void title;
		void description;
		void acceptLabel;
		void rejectLabel;
		void customizeLabel;
		void languageSelector;
		void selectedLanguages;
		void translationsJSONNow;
		void cookieTheme;
		void floatingTrigger;
		void position;
		void ccpaEnabled;
		void ccpaUrl;
		bumpPreview();
	});
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">Cookies</h1>
		<p class="text-[13px] text-text-secondary">
			Same-origin cookie banner. Auto-styled to your brand. Proofs land in your own database.
		</p>
	</header>

	{#if loading}
		<div class="mt-8 flex items-center justify-center py-12">
			<Spinner />
		</div>
	{:else}
		<div class="mt-8 grid grid-cols-1 gap-5 lg:grid-cols-[2fr_1fr]">
			<div class="flex flex-col gap-5">
				<Card padding="md">
					<div class="flex items-center justify-between gap-4">
						<div class="flex flex-col">
							<span class="text-[13px] text-text-primary">Enable cookie banner</span>
							<span class="text-[12px] text-text-muted">
								Ships the embedded widget on every page. Same-origin, no third-party fetch.
							</span>
						</div>
						<Switch bind:checked={enabled} ariaLabel="Enable cookie banner" />
					</div>
				</Card>

				<Card padding="md">
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Banner copy
					</h2>
					<div class="mt-3 flex flex-col gap-3">
						<Input
							label="Title (optional)"
							placeholder="Vi använder kakor"
							hint="Overrides the widget's translated banner heading. Leave empty for the locale default."
							bind:value={title}
						/>
						<Textarea
							rows={3}
							placeholder="Vi använder kakor för att…"
							bind:value={description}
						/>
					</div>
				</Card>

				<Card padding="md">
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Button labels
					</h2>
					<p class="mt-2 text-[12px] text-text-muted">
						Optional overrides. Leave any field empty to use the translated default for the site
						language.
					</p>
					<div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-3">
						<Input label="Accept" placeholder="Godkänn alla" bind:value={acceptLabel} />
						<Input label="Reject" placeholder="Avvisa alla" bind:value={rejectLabel} />
						<Input label="Save preferences" placeholder="Spara inställningar" bind:value={customizeLabel} />
					</div>
				</Card>

				<Card padding="md">
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Position
					</h2>
					<div class="mt-3 flex gap-2">
						{#each ['bottom', 'top', 'center'] as p}
							<label class="flex flex-1 cursor-pointer items-center justify-center gap-2 rounded border px-3 py-2 text-[13px] transition-colors {position === p ? 'border-accent text-text-primary' : 'border-border-light text-text-muted hover:text-text-primary'}">
								<input type="radio" class="sr-only" value={p} bind:group={position} />
								<span class="capitalize">{p}</span>
							</label>
						{/each}
					</div>
				</Card>

				<Card padding="md">
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Categories
					</h2>
					<p class="mt-2 text-[12px] text-text-muted">
						"Necessary" is always required. Toggle off any category you don't use to drop it from
						the banner entirely.
					</p>
					<div class="mt-3 flex flex-col gap-2.5">
						<div class="flex items-center justify-between gap-4">
							<span class="text-[13px] text-text-primary">Analytics</span>
							<Switch bind:checked={catAnalytics} ariaLabel="Offer analytics category" />
						</div>
						<div class="flex items-center justify-between gap-4">
							<span class="text-[13px] text-text-primary">Marketing</span>
							<Switch bind:checked={catMarketing} ariaLabel="Offer marketing category" />
						</div>
						<div class="flex items-center justify-between gap-4">
							<span class="text-[13px] text-text-primary">Preferences</span>
							<Switch bind:checked={catPreferences} ariaLabel="Offer preferences category" />
						</div>
					</div>
				</Card>

				<Card padding="md">
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Compliance &amp; UX
					</h2>
					<p class="mt-2 text-[12px] text-text-muted">
						Settings that affect IMY / GDPR / CCPA compliance and the visitor experience. Defaults
						are safe; touch these only when policy changes or the visitor reports a problem.
					</p>

					<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
						<div>
							<label class="text-[12px] text-text-primary" for="cookie-expiry-days">
								Consent expiry (days)
							</label>
							<input
								id="cookie-expiry-days"
								type="number"
								min="0"
								max="365"
								class="mt-1 w-full rounded border border-border-light bg-transparent px-2 py-1.5 text-[13px] text-text-primary"
								bind:value={cookieExpiryDays}
							/>
							<p class="mt-1 text-[11px] text-text-muted">
								IMY caps consent at 12 months. Default 365. Set 0 to use the widget default.
							</p>
						</div>

						<div>
							<label class="text-[12px] text-text-primary" for="cookie-revision">
								Policy revision
							</label>
							<input
								id="cookie-revision"
								type="number"
								min="0"
								max="9999"
								class="mt-1 w-full rounded border border-border-light bg-transparent px-2 py-1.5 text-[13px] text-text-primary"
								bind:value={cookieRevision}
							/>
							<p class="mt-1 text-[11px] text-text-muted">
								Increment when your privacy policy or category list changes. Forces every visitor
								to re-consent.
							</p>
						</div>

						<div>
							<label class="text-[12px] text-text-primary" for="cookie-theme">Theme</label>
							<select
								id="cookie-theme"
								class="mt-1 w-full rounded border border-border-light bg-transparent px-2 py-1.5 text-[13px] text-text-primary"
								bind:value={cookieTheme}
							>
								<option value="light">Light</option>
								<option value="dark">Dark</option>
								<option value="auto">Auto (follow OS)</option>
							</select>
							<p class="mt-1 text-[11px] text-text-muted">
								Brand colors still apply on top of the theme.
							</p>
						</div>

						<div>
							<label class="text-[12px] text-text-primary" for="floating-trigger">
								Floating reopen trigger
							</label>
							<select
								id="floating-trigger"
								class="mt-1 w-full rounded border border-border-light bg-transparent px-2 py-1.5 text-[13px] text-text-primary"
								bind:value={floatingTrigger}
							>
								<option value="left">Bottom-left</option>
								<option value="right">Bottom-right</option>
								<option value="off">Off (only manual link)</option>
							</select>
						</div>
					</div>

					<div class="mt-4 flex items-center justify-between gap-4">
						<div class="flex flex-col">
							<span class="text-[13px] text-text-primary">Show language selector</span>
							<span class="text-[12px] text-text-muted">
								In-banner dropdown so visitors on the wrong locale can switch the modal language.
							</span>
						</div>
						<Switch bind:checked={languageSelector} ariaLabel="Show language selector in banner" />
					</div>

					{#if languageSelector}
						<div class="mt-4">
							<div class="flex items-baseline justify-between gap-4">
								<span class="text-[13px] text-text-primary">Languages in dropdown</span>
								<span class="text-[11px] text-text-muted">
									{selectedLanguages.length === 0
										? 'All 13 languages'
										: `${selectedLanguages.length} selected`}
								</span>
							</div>
							<p class="mt-1 text-[12px] text-text-muted">
								Pick which languages appear in the cookie banner's language switcher. Leave
								all unchecked to show every built-in language. The site's own language
								({data.site.lang}) is always available.
							</p>
							<div class="mt-2 flex flex-wrap gap-1.5">
								{#each SUPPORTED_LANGUAGES as { code, label }}
									{@const checked = selectedLanguages.includes(code)}
									<label
										class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-[12px] cursor-pointer select-none transition-colors {checked
											? 'border-text-primary bg-text-primary text-bg-base'
											: 'border-border-light bg-bg-elevated text-text-secondary hover:border-text-muted'}"
									>
										<input
											type="checkbox"
											class="sr-only"
											{checked}
											onchange={(e) => {
												const ch = (e.currentTarget as HTMLInputElement).checked;
												if (ch && !selectedLanguages.includes(code)) {
													selectedLanguages = [...selectedLanguages, code];
												} else if (!ch) {
													selectedLanguages = selectedLanguages.filter((c) => c !== code);
												}
											}}
										/>
										<span class="font-mono text-[10px] uppercase opacity-70">{code}</span>
										<span>{label}</span>
									</label>
								{/each}
							</div>
						</div>
					{/if}

					{#if languageSelector}
						<div class="mt-6 border-t border-border-light pt-4">
							<div class="flex items-baseline justify-between gap-4">
								<span class="text-[13px] text-text-primary">Per-language overrides</span>
								<span class="text-[11px] text-text-muted">
									{Object.keys(translations).length === 0
										? 'Built-in translations'
										: `${Object.keys(translations).length} customised`}
								</span>
							</div>
							<p class="mt-1 text-[12px] text-text-muted">
								Override the banner copy for a specific language. Empty fields fall back to the
								widget's built-in translation, so you only need to fill in what you want to change.
							</p>

							<div class="mt-3 flex flex-wrap gap-1.5">
								{#each (selectedLanguages.length > 0 ? selectedLanguages : SUPPORTED_LANGUAGES.map((l) => l.code)) as code}
									{@const langInfo = SUPPORTED_LANGUAGES.find((l) => l.code === code)}
									{@const isOpen = editingLang === code}
									{@const hasOverrides = !!translations[code] && Object.values(translations[code]).some((v) => v)}
									<button
										type="button"
										class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-[12px] cursor-pointer transition-colors {isOpen
											? 'border-text-primary bg-text-primary text-bg-base'
											: hasOverrides
											? 'border-text-primary bg-bg-elevated text-text-primary'
											: 'border-border-light bg-bg-elevated text-text-secondary hover:border-text-muted'}"
										onclick={() => (editingLang = isOpen ? '' : code)}
									>
										<span class="font-mono text-[10px] uppercase opacity-70">{code}</span>
										<span>{langInfo?.label ?? code}</span>
										{#if hasOverrides && !isOpen}
											<span class="h-1.5 w-1.5 rounded-full bg-text-primary"></span>
										{/if}
									</button>
								{/each}
							</div>

							{#if editingLang}
								{@const langInfo = SUPPORTED_LANGUAGES.find((l) => l.code === editingLang)}
								<div class="mt-4 rounded-lg border border-border-light bg-bg-elevated p-4">
									<div class="mb-3 flex items-center justify-between gap-4">
										<div class="flex flex-col">
											<span class="text-[13px] text-text-primary">{langInfo?.label ?? editingLang}</span>
											<span class="text-[11px] text-text-muted font-mono uppercase">{editingLang}</span>
										</div>
										<button
											type="button"
											class="text-[11px] text-text-muted hover:text-red-500"
											onclick={() => clearLang(editingLang)}
										>
											Clear all
										</button>
									</div>
									<div class="flex flex-col gap-3">
										<Input
											label="Title"
											placeholder="Banner heading in {langInfo?.label ?? editingLang}"
											value={tField(editingLang, 'title')}
											oninput={(e) => setTField(editingLang, 'title', (e.currentTarget as HTMLInputElement).value)}
										/>
										<Textarea
											label="Description"
											rows={3}
											placeholder="Banner body text"
											value={tField(editingLang, 'description')}
											oninput={(e) => setTField(editingLang, 'description', (e.currentTarget as HTMLTextAreaElement).value)}
										/>
										<div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
											<Input
												label="Accept"
												value={tField(editingLang, 'accept')}
												oninput={(e) => setTField(editingLang, 'accept', (e.currentTarget as HTMLInputElement).value)}
											/>
											<Input
												label="Reject"
												value={tField(editingLang, 'reject')}
												oninput={(e) => setTField(editingLang, 'reject', (e.currentTarget as HTMLInputElement).value)}
											/>
											<Input
												label="Save preferences"
												value={tField(editingLang, 'customize')}
												oninput={(e) => setTField(editingLang, 'customize', (e.currentTarget as HTMLInputElement).value)}
											/>
										</div>
									</div>
								</div>
							{/if}
						</div>
					{/if}

					<div class="mt-4 flex items-center justify-between gap-4">
						<div class="flex flex-col">
							<span class="text-[13px] text-text-primary">CCPA "Do Not Sell" link</span>
							<span class="text-[12px] text-text-muted">
								Surfaces a link in the banner footer. Required if you serve California visitors
								and use third-party trackers.
							</span>
						</div>
						<Switch bind:checked={ccpaEnabled} ariaLabel="Enable CCPA Do Not Sell link" />
					</div>

					{#if ccpaEnabled}
						<div class="mt-3">
							<Input
								label="CCPA disclosure URL"
								placeholder="/privacy#ccpa or https://example.com/do-not-sell"
								hint="Absolute URL or root-relative path to your disclosure page."
								bind:value={ccpaUrl}
							/>
						</div>
					{/if}
				</Card>

				<Card padding="md">
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Cookies declared
					</h2>
					<p class="mt-2 text-[12px] text-text-muted">
						Per-cookie metadata for the extended-settings table inside the consent modal. Rows
						labeled <span class="rounded bg-bg-surface px-1.5 py-0.5 text-[11px] text-text-secondary">Auto</span>
						come from your enabled trackers and language settings; they always apply unless you
						override them. Click <span class="text-text-primary">Override</span> on any auto row to
						edit the strings (handy when translating to Swedish).
					</p>

					<div class="mt-4 flex flex-col gap-5">
						{#each CATEGORY_ORDER as cat}
							{@const presetRows = presetRowsFor(cat)}
							{@const userRows = userRowsFor(cat)}
							<div class="rounded-lg border border-border-light p-3">
								<div class="flex items-center justify-between">
									<h3 class="text-[12px] font-medium text-text-primary">{CATEGORY_LABELS[cat]}</h3>
									<Button variant="ghost" onclick={() => addDeclaration(cat)}>+ Add cookie</Button>
								</div>

								{#if presetRows.length > 0 || userRows.length > 0}
									<div class="mt-3 overflow-x-auto">
										<table class="w-full text-[12px]">
											<thead>
												<tr class="text-left text-text-muted">
													<th class="pb-1.5 pr-3 font-normal">Cookie</th>
													<th class="pb-1.5 pr-3 font-normal">Provider</th>
													<th class="pb-1.5 pr-3 font-normal">Purpose</th>
													<th class="pb-1.5 pr-3 font-normal">Expiry</th>
													<th class="pb-1.5 font-normal w-16"></th>
												</tr>
											</thead>
											<tbody class="align-top">
												{#each presetRows as p}
													{@const overridden = isOverridden(p)}
													<tr class="border-t border-border-light/60 {overridden ? 'opacity-40' : ''}">
														<td class="py-2 pr-3">
															<div class="flex items-center gap-2">
																<code class="font-mono text-[11px] text-text-primary">{p.name}</code>
																<span class="rounded bg-bg-surface px-1.5 py-0.5 text-[10px] text-text-secondary">Auto</span>
															</div>
														</td>
														<td class="py-2 pr-3 text-text-secondary">{p.provider ?? '-'}</td>
														<td class="py-2 pr-3 text-text-secondary">{p.purpose ?? '-'}</td>
														<td class="py-2 pr-3 text-text-secondary">{p.expiry ?? '-'}</td>
														<td class="py-2">
															{#if !overridden}
																<button type="button" class="text-[11px] text-accent hover:underline" onclick={() => overridePreset(p)}>Override</button>
															{:else}
																<span class="text-[11px] text-text-muted">overridden</span>
															{/if}
														</td>
													</tr>
												{/each}
												{#each userRows as { row, idx } (idx)}
													<tr class="border-t border-border-light/60">
														<td class="py-2 pr-3">
															<input
																type="text"
																class="w-full rounded border border-border-light bg-transparent px-2 py-1 font-mono text-[11px] text-text-primary"
																placeholder="_ga"
																bind:value={row.name}
															/>
														</td>
														<td class="py-2 pr-3">
															<input
																type="text"
																class="w-full rounded border border-border-light bg-transparent px-2 py-1 text-[12px] text-text-primary"
																placeholder="Google"
																bind:value={row.provider}
															/>
														</td>
														<td class="py-2 pr-3">
															<input
																type="text"
																class="w-full rounded border border-border-light bg-transparent px-2 py-1 text-[12px] text-text-primary"
																placeholder="Distinguishes unique users"
																bind:value={row.purpose}
															/>
														</td>
														<td class="py-2 pr-3">
															<input
																type="text"
																class="w-full rounded border border-border-light bg-transparent px-2 py-1 text-[12px] text-text-primary"
																placeholder="2 years"
																bind:value={row.expiry}
															/>
														</td>
														<td class="py-2">
															<button
																type="button"
																class="text-[11px] text-text-muted hover:text-red-500"
																aria-label="Remove cookie"
																onclick={() => removeDeclaration(idx)}
															>×</button>
														</td>
													</tr>
												{/each}
											</tbody>
										</table>
									</div>
								{:else}
									<p class="mt-3 text-[11px] italic text-text-muted">
										No cookies declared. Add one to populate the extended-settings table for this
										category.
									</p>
								{/if}
							</div>
						{/each}
					</div>
				</Card>

				<div
					class="sticky bottom-0 -mx-6 flex items-center justify-between gap-3 border-t border-border-light bg-bg-surface/85 px-6 py-3 backdrop-blur"
				>
					<span class="text-[12px] text-text-muted">
						{dirty ? 'Unsaved changes.' : 'Up to date.'}
					</span>
					<div class="flex items-center gap-2">
						<Button variant="ghost" onclick={discard} disabled={!dirty || saving}>Discard</Button>
						<Button variant="primary" onclick={save} loading={saving} disabled={!dirty}>Save</Button>
					</div>
				</div>
			</div>

			<div class="flex flex-col gap-5">
				<Card padding="md">
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Live preview
					</h2>
					<p class="mt-1 text-[11px] text-text-muted">
						The actual CookieProof widget rendered with this site's branding, copy,
						and cookie tables. Updates as you edit (debounced ~350ms).
					</p>
					<div class="mt-3 overflow-hidden rounded-lg border border-border-light">
						<iframe
							src={previewSrc}
							title="Cookie banner preview"
							class="block h-[640px] w-full bg-bg-elevated"
							sandbox="allow-scripts allow-same-origin"
						></iframe>
					</div>
					<p class="mt-3 text-[11px] text-text-muted">
						Reject + Accept share the primary color (IMY 2026 equal prominence). Save preferences
						uses the secondary brand color. Language selector and privacy policy link wire
						automatically when configured. Rebuild the site to publish.
					</p>
				</Card>

				<Card padding="md">
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						How it works
					</h2>
					<ul class="mt-3 flex flex-col gap-2 text-[12px] text-text-muted">
						<li>
							<span class="text-text-primary">Same-origin asset.</span> Widget served from
							<code class="font-mono text-[11px]">/_ccb.&lt;hash&gt;.js</code> on your domain.
						</li>
						<li>
							<span class="text-text-primary">Inline config.</span> Branding colors flow into
							CSS vars at build time. No remote fetch.
						</li>
						<li>
							<span class="text-text-primary">Proofs in your DB.</span> Consent records POST to
							<code class="font-mono text-[11px]">/t/consent</code> and land in
							<code class="font-mono text-[11px]">consent_records</code>.
						</li>
						<li>
							<span class="text-text-primary">GPC respected.</span> Visitors with Global Privacy
							Control set are auto-rejected and the banner is suppressed.
						</li>
					</ul>
				</Card>
			</div>
		</div>
	{/if}
</div>

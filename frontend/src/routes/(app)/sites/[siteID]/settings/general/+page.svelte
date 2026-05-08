<script lang="ts">
	import * as sitesApi from '$lib/api/sites';
	import * as settingsApi from '$lib/api/settings';
	import { ApiError } from '$lib/api/client';
	import Input from '$lib/components/ui/Input.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import { setSite } from '$lib/stores/currentSite.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { categoryMap } from '$lib/settings/nginxPreview';
	import type { Site } from '$lib/api/types';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);

	const langOptions = [
		{ value: 'en', label: 'English (en)' },
		{ value: 'sv', label: 'Swedish (sv)' },
		{ value: 'de', label: 'German (de)' },
		{ value: 'fr', label: 'French (fr)' },
		{ value: 'es', label: 'Spanish (es)' },
		{ value: 'it', label: 'Italian (it)' },
		{ value: 'no', label: 'Norwegian (no)' },
		{ value: 'da', label: 'Danish (da)' },
		{ value: 'fi', label: 'Finnish (fi)' },
		{ value: 'nl', label: 'Dutch (nl)' }
	];

	let loading = $state(true);
	let saving = $state(false);

	let name = $state('');
	let domain = $state('');
	let lang = $state('en');

	let metaTitleTemplate = $state('');
	let metaDescriptionTemplate = $state('');
	let canonicalBase = $state('');
	let additionalLangs = $state('');
	let hreflangStrategy = $state<'path' | 'subdomain' | 'off'>('path');

	const hreflangOptions = [
		{ value: 'path', label: 'Path-based (/sv/about, /de/about)' },
		{ value: 'subdomain', label: 'Subdomain (sv.example.com/about)' },
		{ value: 'off', label: 'Off (no hreflang emission)' }
	];

	const strategyHint = $derived.by(() => {
		switch (hreflangStrategy) {
			case 'path':
				return 'Recommended. Each language sits at /<lang>/<slug>; default language at root. Atomicsite emits hreflang automatically when a page has counterparts in other locales (e.g. /about + /sv/about both published). One domain, one cert, no DNS work.';
			case 'subdomain':
				return 'For sites already running sv.example.com / de.example.com separately. Atomicsite trusts your additional_langs list; you handle the DNS + TLS per subdomain. Hreflang URLs use the <lang>.<host> pattern.';
			case 'off':
				return 'Disable hreflang emission entirely. Use only if you manage hreflang from a custom layout or your site is single-language and you want to be explicit.';
		}
	});

	let initial = $state({
		name: '',
		domain: '',
		lang: 'en',
		metaTitleTemplate: '',
		metaDescriptionTemplate: '',
		canonicalBase: '',
		additionalLangs: '',
		hreflangStrategy: 'path' as 'path' | 'subdomain' | 'off'
	});

	async function load() {
		loading = true;
		try {
			name = data.site.name;
			domain = data.site.domain;
			lang = data.site.lang || 'en';

			const [seo, general] = await Promise.all([
				settingsApi.listByCategory(siteID, 'seo'),
				settingsApi.listByCategory(siteID, 'general')
			]);
			const seoMap = categoryMap(seo);
			const genMap = categoryMap(general);

			metaTitleTemplate = seoMap.meta_title_template || '';
			metaDescriptionTemplate = seoMap.meta_description_template || '';
			canonicalBase = seoMap.canonical_base || '';
			additionalLangs = genMap.additional_langs || '';
			const strat = (seoMap.hreflang_strategy || 'path') as
				| 'path'
				| 'subdomain'
				| 'off';
			hreflangStrategy = ['path', 'subdomain', 'off'].includes(strat) ? strat : 'path';

			initial = {
				name,
				domain,
				lang,
				metaTitleTemplate,
				metaDescriptionTemplate,
				canonicalBase,
				additionalLangs,
				hreflangStrategy
			};
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to load settings.');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	const dirty = $derived(
		name !== initial.name ||
			domain !== initial.domain ||
			lang !== initial.lang ||
			metaTitleTemplate !== initial.metaTitleTemplate ||
			metaDescriptionTemplate !== initial.metaDescriptionTemplate ||
			canonicalBase !== initial.canonicalBase ||
			additionalLangs !== initial.additionalLangs ||
			hreflangStrategy !== initial.hreflangStrategy
	);

	function discard() {
		name = initial.name;
		domain = initial.domain;
		lang = initial.lang;
		metaTitleTemplate = initial.metaTitleTemplate;
		metaDescriptionTemplate = initial.metaDescriptionTemplate;
		canonicalBase = initial.canonicalBase;
		additionalLangs = initial.additionalLangs;
		hreflangStrategy = initial.hreflangStrategy;
	}

	// Live preview of how the meta-title template renders with a sample
	// page. Uses the same {token} substitution the builder applies; empty
	// tokens collapse along with adjacent separators.
	function expandPreview(tpl: string, fallback: string): string {
		if (!tpl.trim()) return fallback;
		const vars: Record<string, string> = {
			'{page_title}': 'About us',
			'{page_description}': 'Who we are and what we do',
			'{site_name}': name || 'Site name',
			'{lang}': lang || 'en',
			'{separator}': '|'
		};
		let out = tpl;
		for (const [k, v] of Object.entries(vars)) {
			out = out.split(k).join(v);
		}
		// Collapse empty separator runs left over by missing tokens.
		out = out.replace(/\s*\|\s*\|\s*/g, ' | ');
		out = out.replace(/^\s*\|\s*|\s*\|\s*$/g, '');
		out = out.replace(/\s+/g, ' ').trim();
		return out || fallback;
	}

	const titlePreview = $derived(expandPreview(metaTitleTemplate, 'About us'));
	const descPreview = $derived(
		expandPreview(metaDescriptionTemplate, 'Who we are and what we do')
	);

	async function save() {
		if (saving || !dirty) return;
		saving = true;
		try {
			const sitePatch: Record<string, string> = {};
			if (name !== initial.name) sitePatch.name = name;
			if (domain !== initial.domain) sitePatch.domain = domain;
			if (lang !== initial.lang) sitePatch.lang = lang;

			let updatedSite: Site | null = null;
			if (Object.keys(sitePatch).length > 0) {
				updatedSite = await sitesApi.update(siteID, sitePatch);
				setSite(updatedSite);
			}

			const items: settingsApi.SettingUpsertInput[] = [
				{ category: 'seo', key: 'meta_title_template', value: metaTitleTemplate },
				{ category: 'seo', key: 'meta_description_template', value: metaDescriptionTemplate },
				{ category: 'seo', key: 'canonical_base', value: canonicalBase },
				{ category: 'seo', key: 'hreflang_strategy', value: hreflangStrategy },
				{ category: 'general', key: 'additional_langs', value: additionalLangs },
				{ category: 'general', key: 'default_lang', value: lang }
			];
			await settingsApi.bulkUpsert(siteID, items);

			initial = {
				name,
				domain,
				lang,
				metaTitleTemplate,
				metaDescriptionTemplate,
				canonicalBase,
				additionalLangs,
				hreflangStrategy
			};
			toast.success('General settings saved.');
		} catch (err) {
			const msg = err instanceof ApiError ? err.message : 'Failed to save settings.';
			toast.error(msg);
		} finally {
			saving = false;
		}
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
			General
		</h1>
		<p class="text-[13px] text-text-secondary">
			Site name, languages, and meta-tag defaults. Custom hostnames + TLS live in
			<a href="/sites/{siteID}/settings/domains" class="text-accent underline-offset-2 hover:underline">Domains</a>;
			where the built artifact ships to is in
			<a href="/sites/{siteID}/settings/deployment" class="text-accent underline-offset-2 hover:underline">Deployment</a>.
		</p>
	</header>

	{#if loading}
		<div class="mt-8 flex items-center justify-center py-12">
			<Spinner />
		</div>
	{:else}
		<div class="mt-8 flex flex-col gap-5">
			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Identity</h2>
				<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
					<Input label="Site name" bind:value={name} />
					<Input
						label="Default canonical hostname"
						placeholder="example.com"
						hint="Legacy fallback. If you've added a hostname in Domains and marked it canonical, that wins."
						bind:value={domain}
					/>
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Languages and hreflang
				</h2>
				<p class="mt-2 text-[12px] text-text-muted">
					Atomicsite optimizes for path-based multi-language by default
					(/sv/about, /de/about). Subdomain mode is opt-in for sites that
					already run sv.example.com. TLD-per-locale (example.se,
					example.de) is handled as separate sites linked together; that
					setup lives outside this page.
				</p>
				<div class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
					<div class="flex flex-col gap-1.5">
						<label for="lang" class="text-[12px] font-medium text-text-secondary">
							Default language
						</label>
						<Select options={langOptions} bind:value={lang} />
					</div>
					<Input
						label="Additional languages"
						placeholder="sv,de,fr"
						hint="Comma-separated language codes (sv, de, fr). Pages whose slug starts with /<lang>/ are treated as that language's locale."
						bind:value={additionalLangs}
					/>
					<div class="flex flex-col gap-1.5 sm:col-span-2">
						<span class="text-[12px] font-medium text-text-secondary">
							Hreflang strategy
						</span>
						<Select
							options={hreflangOptions}
							value={hreflangStrategy}
							onchange={(v) => (hreflangStrategy = v as 'path' | 'subdomain' | 'off')}
						/>
						<p class="text-[11px] text-text-muted">{strategyHint}</p>
					</div>
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Meta defaults
				</h2>
				<p class="mt-2 text-[12px] text-text-muted">
					Templates apply to every page that doesn't override its own meta.
					Leave empty to use the page-level value verbatim. Tokens:
					<code class="font-mono text-[11px]">{'{page_title}'}</code>,
					<code class="font-mono text-[11px]">{'{site_name}'}</code>,
					<code class="font-mono text-[11px]">{'{lang}'}</code>,
					<code class="font-mono text-[11px]">{'{page_description}'}</code>,
					<code class="font-mono text-[11px]">{'{separator}'}</code>. Empty
					tokens drop with adjacent
					<code class="font-mono text-[11px]">|</code>
					so a missing site_name doesn't leave a trailing pipe.
				</p>
				<div class="mt-4 flex flex-col gap-4">
					<div class="flex flex-col gap-1.5">
						<Input
							label="Meta title template"
							placeholder={'{page_title} | {site_name}'}
							bind:value={metaTitleTemplate}
						/>
						<p class="rounded-md border border-border-light bg-bg-elevated/50 px-3 py-2 font-mono text-[11.5px] text-text-secondary">
							Preview: <span class="text-text-primary">{titlePreview}</span>
						</p>
					</div>
					<div class="flex flex-col gap-1.5">
						<Input
							label="Meta description template"
							placeholder={'{page_description}'}
							bind:value={metaDescriptionTemplate}
						/>
						<p class="rounded-md border border-border-light bg-bg-elevated/50 px-3 py-2 font-mono text-[11.5px] text-text-secondary">
							Preview: <span class="text-text-primary">{descPreview}</span>
						</p>
					</div>
					<Input
						label="Canonical base"
						placeholder="https://example.com"
						hint="Override the default URL prefix (https://your primary domain). Useful when the public URL differs from the build's domain (CDN, sub-path proxy, staging vs prod)."
						bind:value={canonicalBase}
					/>
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
					<Button variant="primary" onclick={save} loading={saving} disabled={!dirty}>
						Save
					</Button>
				</div>
			</div>
		</div>
	{/if}
</div>

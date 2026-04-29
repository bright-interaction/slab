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
	let domainAliases = $state('');
	let additionalLangs = $state('');

	let initial = $state({
		name: '',
		domain: '',
		lang: 'en',
		metaTitleTemplate: '',
		metaDescriptionTemplate: '',
		canonicalBase: '',
		domainAliases: '',
		additionalLangs: ''
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
			domainAliases = genMap.domain_aliases || '';
			additionalLangs = genMap.additional_langs || '';

			initial = {
				name,
				domain,
				lang,
				metaTitleTemplate,
				metaDescriptionTemplate,
				canonicalBase,
				domainAliases,
				additionalLangs
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
			domainAliases !== initial.domainAliases ||
			additionalLangs !== initial.additionalLangs
	);

	function discard() {
		name = initial.name;
		domain = initial.domain;
		lang = initial.lang;
		metaTitleTemplate = initial.metaTitleTemplate;
		metaDescriptionTemplate = initial.metaDescriptionTemplate;
		canonicalBase = initial.canonicalBase;
		domainAliases = initial.domainAliases;
		additionalLangs = initial.additionalLangs;
	}

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
				{ category: 'general', key: 'domain_aliases', value: domainAliases },
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
				domainAliases,
				additionalLangs
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
			Site identity, primary domain, language and meta defaults.
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
					<Input label="Primary domain" placeholder="example.com" bind:value={domain} />
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Languages
				</h2>
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
						hint="Comma-separated list of language codes."
						bind:value={additionalLangs}
					/>
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Meta defaults
				</h2>
				<div class="mt-4 flex flex-col gap-4">
					<Input
						label="Meta title template"
						placeholder={'{page_title} | {site_name}'}
						hint={'Tokens. {page_title}, {site_name}, {lang}.'}
						bind:value={metaTitleTemplate}
					/>
					<Input
						label="Meta description template"
						placeholder={'{page_description}'}
						hint="Falls back to page-level description if empty."
						bind:value={metaDescriptionTemplate}
					/>
					<Input
						label="Canonical base"
						placeholder="https://example.com"
						hint="Used to build canonical URLs at build time."
						bind:value={canonicalBase}
					/>
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Aliases</h2>
				<div class="mt-4">
					<Input
						label="Domain aliases"
						placeholder="www.example.com, example.org"
						hint="Comma-separated. Each alias 301-redirects to the primary domain."
						bind:value={domainAliases}
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

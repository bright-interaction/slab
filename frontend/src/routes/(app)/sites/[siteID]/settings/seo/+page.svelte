<script lang="ts">
	import * as settingsApi from '$lib/api/settings';
	import * as mediaApi from '$lib/api/media';
	import { ApiError } from '$lib/api/client';
	import Input from '$lib/components/ui/Input.svelte';
	import Switch from '$lib/components/ui/Switch.svelte';
	import Textarea from '$lib/components/ui/Textarea.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Spinner from '$lib/components/ui/Spinner.svelte';
	import MediaPicker from '$lib/components/media/MediaPicker.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { categoryMap } from '$lib/settings/nginxPreview';
	import type { Site, Medium } from '$lib/api/types';
	import { Trash2, Plus } from 'lucide-svelte';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);

	function toBool(v: string | undefined): boolean {
		if (!v) return false;
		const s = v.toLowerCase();
		return s === '1' || s === 'true' || s === 'yes' || s === 'on';
	}

	type HreflangRow = { lang: string; hreflang_code: string };

	let loading = $state(true);
	let saving = $state(false);

	let metaTitleTemplate = $state('');
	let metaDescriptionTemplate = $state('');
	let canonicalBase = $state('');
	let ogDefaultImageId = $state('');
	let ogDefaultImage = $state<Medium | null>(null);
	let robotsExtra = $state('');
	let llmsTxt = $state('');
	let sitemapEnabled = $state(true);
	let hreflangRows = $state<HreflangRow[]>([]);

	let pickerOpen = $state(false);

	type State = {
		metaTitleTemplate: string;
		metaDescriptionTemplate: string;
		canonicalBase: string;
		ogDefaultImageId: string;
		robotsExtra: string;
		llmsTxt: string;
		sitemapEnabled: boolean;
		hreflangJson: string;
	};

	let initial: State = $state({
		metaTitleTemplate: '',
		metaDescriptionTemplate: '',
		canonicalBase: '',
		ogDefaultImageId: '',
		robotsExtra: '',
		llmsTxt: '',
		sitemapEnabled: true,
		hreflangJson: '[]'
	});

	function parseHreflang(raw: string): HreflangRow[] {
		if (!raw.trim()) return [];
		try {
			const parsed = JSON.parse(raw);
			if (!Array.isArray(parsed)) return [];
			return parsed
				.filter(
					(r): r is HreflangRow =>
						r &&
						typeof r === 'object' &&
						typeof r.lang === 'string' &&
						typeof r.hreflang_code === 'string'
				)
				.map((r) => ({ lang: r.lang, hreflang_code: r.hreflang_code }));
		} catch {
			return [];
		}
	}

	async function load() {
		loading = true;
		try {
			const rows = await settingsApi.listByCategory(siteID, 'seo');
			const m = categoryMap(rows);

			metaTitleTemplate = m.meta_title_template || '';
			metaDescriptionTemplate = m.meta_description_template || '';
			canonicalBase = m.canonical_base || '';
			ogDefaultImageId = m.og_default_image_id || '';
			robotsExtra = m.robots_txt_extra || '';
			llmsTxt = m.llms_txt || '';
			sitemapEnabled = m.sitemap_enabled ? toBool(m.sitemap_enabled) : true;

			const hreflangJson = m.hreflang_map || '[]';
			hreflangRows = parseHreflang(hreflangJson);

			if (ogDefaultImageId) {
				try {
					ogDefaultImage = await mediaApi.get(siteID, ogDefaultImageId);
				} catch {
					ogDefaultImage = null;
				}
			}

			initial = {
				metaTitleTemplate,
				metaDescriptionTemplate,
				canonicalBase,
				ogDefaultImageId,
				robotsExtra,
				llmsTxt,
				sitemapEnabled,
				hreflangJson: JSON.stringify(hreflangRows)
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

	const hreflangJsonNow = $derived(JSON.stringify(hreflangRows));

	const dirty = $derived(
		metaTitleTemplate !== initial.metaTitleTemplate ||
			metaDescriptionTemplate !== initial.metaDescriptionTemplate ||
			canonicalBase !== initial.canonicalBase ||
			ogDefaultImageId !== initial.ogDefaultImageId ||
			robotsExtra !== initial.robotsExtra ||
			llmsTxt !== initial.llmsTxt ||
			sitemapEnabled !== initial.sitemapEnabled ||
			hreflangJsonNow !== initial.hreflangJson
	);

	function discard() {
		metaTitleTemplate = initial.metaTitleTemplate;
		metaDescriptionTemplate = initial.metaDescriptionTemplate;
		canonicalBase = initial.canonicalBase;
		ogDefaultImageId = initial.ogDefaultImageId;
		robotsExtra = initial.robotsExtra;
		llmsTxt = initial.llmsTxt;
		sitemapEnabled = initial.sitemapEnabled;
		hreflangRows = parseHreflang(initial.hreflangJson);
	}

	function addHreflang() {
		hreflangRows = [...hreflangRows, { lang: '', hreflang_code: '' }];
	}

	function removeHreflang(i: number) {
		hreflangRows = hreflangRows.filter((_, idx) => idx !== i);
	}

	function pickImage(m: Medium) {
		ogDefaultImageId = m.id;
		ogDefaultImage = m;
	}

	function clearImage() {
		ogDefaultImageId = '';
		ogDefaultImage = null;
	}

	async function save() {
		if (saving || !dirty) return;
		saving = true;
		try {
			const items: settingsApi.SettingUpsertInput[] = [
				{ category: 'seo', key: 'meta_title_template', value: metaTitleTemplate },
				{ category: 'seo', key: 'meta_description_template', value: metaDescriptionTemplate },
				{ category: 'seo', key: 'canonical_base', value: canonicalBase },
				{ category: 'seo', key: 'og_default_image_id', value: ogDefaultImageId },
				{ category: 'seo', key: 'robots_txt_extra', value: robotsExtra },
				{ category: 'seo', key: 'llms_txt', value: llmsTxt },
				{ category: 'seo', key: 'sitemap_enabled', value: sitemapEnabled ? '1' : '0' },
				{ category: 'seo', key: 'hreflang_map', value: hreflangJsonNow }
			];
			await settingsApi.bulkUpsert(siteID, items);

			initial = {
				metaTitleTemplate,
				metaDescriptionTemplate,
				canonicalBase,
				ogDefaultImageId,
				robotsExtra,
				llmsTxt,
				sitemapEnabled,
				hreflangJson: hreflangJsonNow
			};
			toast.success('SEO settings saved.');
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
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">SEO</h1>
		<p class="text-[13px] text-text-secondary">
			Defaults for meta tags, sitemaps, robots.txt and hreflang.
		</p>
	</header>

	{#if loading}
		<div class="mt-8 flex items-center justify-center py-12">
			<Spinner />
		</div>
	{:else}
		<div class="mt-8 flex flex-col gap-5">
			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Meta defaults
				</h2>
				<div class="mt-4 flex flex-col gap-4">
					<Input
						label="Meta title template"
						placeholder={'{page_title} | {site_name}'}
						bind:value={metaTitleTemplate}
					/>
					<Input
						label="Meta description template"
						placeholder={'{page_description}'}
						bind:value={metaDescriptionTemplate}
					/>
					<Input
						label="Canonical base"
						placeholder="https://example.com"
						bind:value={canonicalBase}
					/>
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Default Open Graph image
				</h2>
				<div class="mt-4 flex items-center gap-4">
					{#if ogDefaultImage}
						<div
							class="h-16 w-28 shrink-0 overflow-hidden rounded-lg border border-border-light bg-bg-elevated"
						>
							<img
								src={`/api/sites/${siteID}/media/${ogDefaultImage.id}/file`}
								alt={ogDefaultImage.alt_text || ogDefaultImage.filename}
								class="h-full w-full object-cover"
							/>
						</div>
						<div class="flex flex-col gap-1 min-w-0">
							<p class="text-[13px] text-text-primary truncate">{ogDefaultImage.filename}</p>
							<div class="flex items-center gap-2">
								<Button variant="secondary" size="sm" onclick={() => (pickerOpen = true)}>
									Change
								</Button>
								<Button variant="ghost" size="sm" onclick={clearImage}>Clear</Button>
							</div>
						</div>
					{:else}
						<Button variant="secondary" onclick={() => (pickerOpen = true)}>Choose image</Button>
						<span class="text-[12px] text-text-muted">
							Used when a page has no og:image set.
						</span>
					{/if}
				</div>
			</Card>

			<Card padding="md">
				<div class="flex items-center justify-between">
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Hreflang
					</h2>
					<Button variant="ghost" size="sm" onclick={addHreflang}>
						{#snippet icon()}
							<Plus size={14} strokeWidth={1.75} />
						{/snippet}
						Add row
					</Button>
				</div>
				<div class="mt-4 flex flex-col gap-2">
					{#if hreflangRows.length === 0}
						<p class="text-[12px] text-text-muted">
							No hreflang entries. Add one per supported language or region.
						</p>
					{:else}
						{#each hreflangRows as row, i (i)}
							<div class="grid grid-cols-[1fr_1fr_auto] gap-2 items-center">
								<Input placeholder="en" bind:value={row.lang} />
								<Input placeholder="en-US" bind:value={row.hreflang_code} />
								<button
									type="button"
									class="inline-flex h-9 w-9 items-center justify-center rounded-lg text-text-muted hover:bg-bg-hover hover:text-danger transition-colors"
									aria-label="Remove row"
									onclick={() => removeHreflang(i)}
								>
									<Trash2 size={14} strokeWidth={1.75} />
								</button>
							</div>
						{/each}
					{/if}
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Sitemap and robots
				</h2>
				<div class="mt-4 flex flex-col gap-4">
					<div class="flex items-center justify-between gap-4">
						<div class="flex flex-col">
							<span class="text-[13px] text-text-primary">Generate sitemap.xml</span>
							<span class="text-[12px] text-text-muted">
								Includes all published pages on each build.
							</span>
						</div>
						<Switch bind:checked={sitemapEnabled} />
					</div>
					<Textarea
						label="robots.txt extras"
						rows={4}
						placeholder={`Disallow: /private/\nUser-agent: GPTBot\nDisallow: /`}
						hint="Appended to the auto-generated robots.txt."
						bind:value={robotsExtra}
					/>
					<Textarea
						label="llms.txt content"
						rows={4}
						placeholder="# llms.txt&#10;Brief description for LLM crawlers."
						hint="Optional. Served at /llms.txt for AI crawlers."
						bind:value={llmsTxt}
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

<MediaPicker bind:open={pickerOpen} {siteID} onSelect={pickImage} />

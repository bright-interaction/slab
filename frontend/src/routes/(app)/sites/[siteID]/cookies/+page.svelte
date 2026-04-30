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

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);

	function toBool(v: string | undefined, fallback = false): boolean {
		if (v === undefined || v === '') return fallback;
		const s = v.toLowerCase();
		return s === '1' || s === 'true' || s === 'yes' || s === 'on';
	}

	let loading = $state(true);
	let saving = $state(false);

	let enabled = $state(false);
	let position = $state<'bottom' | 'top' | 'center'>('bottom');
	let title = $state('');
	let description = $state('');
	let catAnalytics = $state(true);
	let catMarketing = $state(true);
	let catPreferences = $state(true);

	type State = {
		enabled: boolean;
		position: 'bottom' | 'top' | 'center';
		title: string;
		description: string;
		catAnalytics: boolean;
		catMarketing: boolean;
		catPreferences: boolean;
	};

	let initial: State = $state({
		enabled: false,
		position: 'bottom',
		title: '',
		description: '',
		catAnalytics: true,
		catMarketing: true,
		catPreferences: true
	});

	async function load() {
		loading = true;
		try {
			const rows = await settingsApi.listByCategory(siteID, 'analytics');
			const m = categoryMap(rows);

			enabled = toBool(m.cookieproof_enabled);
			const pos = (m.cookie_banner_position || 'bottom').toLowerCase();
			position = pos === 'top' || pos === 'center' ? pos : 'bottom';
			title = m.cookie_banner_title || '';
			description = m.cookie_banner_description || '';
			catAnalytics = toBool(m.cookie_cat_analytics, true);
			catMarketing = toBool(m.cookie_cat_marketing, true);
			catPreferences = toBool(m.cookie_cat_preferences, true);

			initial = { enabled, position, title, description, catAnalytics, catMarketing, catPreferences };
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to load cookie settings.');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	const dirty = $derived(
		enabled !== initial.enabled ||
			position !== initial.position ||
			title !== initial.title ||
			description !== initial.description ||
			catAnalytics !== initial.catAnalytics ||
			catMarketing !== initial.catMarketing ||
			catPreferences !== initial.catPreferences
	);

	function discard() {
		enabled = initial.enabled;
		position = initial.position;
		title = initial.title;
		description = initial.description;
		catAnalytics = initial.catAnalytics;
		catMarketing = initial.catMarketing;
		catPreferences = initial.catPreferences;
	}

	function b(v: boolean): string {
		return v ? '1' : '0';
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
				{ category: 'analytics', key: 'cookie_cat_analytics', value: b(catAnalytics) },
				{ category: 'analytics', key: 'cookie_cat_marketing', value: b(catMarketing) },
				{ category: 'analytics', key: 'cookie_cat_preferences', value: b(catPreferences) }
			];
			await settingsApi.bulkUpsert(siteID, items);
			initial = { enabled, position, title, description, catAnalytics, catMarketing, catPreferences };
			toast.success('Cookie banner settings saved. Rebuild to publish.');
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
					<div
						class="mt-3 rounded-lg border border-border-light p-4"
						style:background-color={data.site.bg_color || '#FAFAFA'}
						style:color={data.site.text_color || '#1A1A1A'}
					>
						<div class="text-[15px] font-medium" style:font-family={data.site.font_body}>
							{title || 'Vi använder kakor'}
						</div>
						<p class="mt-1.5 text-[12px] leading-relaxed opacity-80">
							{description || 'Vi använder kakor för att förbättra din upplevelse, mäta trafik och anpassa innehåll. Du kan när som helst ändra dina inställningar.'}
						</p>
						<div class="mt-3 flex gap-2">
							<button
								type="button"
								class="flex-1 rounded px-3 py-2 text-[12px] font-medium"
								style:background-color={data.site.primary_color || '#0066CC'}
								style:color={data.site.on_primary_color || '#FFFFFF'}
							>Acceptera</button>
							<button
								type="button"
								class="flex-1 rounded border px-3 py-2 text-[12px]"
								style:border-color={data.site.border_color || '#E5E7EB'}
							>Avvisa</button>
						</div>
					</div>
					<p class="mt-3 text-[11px] text-text-muted">
						The real banner uses your full brand palette. Rebuild the site to publish changes.
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

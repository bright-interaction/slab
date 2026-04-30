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

	function toBool(v: string | undefined): boolean {
		if (!v) return false;
		const s = v.toLowerCase();
		return s === '1' || s === 'true' || s === 'yes' || s === 'on';
	}

	let loading = $state(true);
	let saving = $state(false);

	let atomicsiteTrackingEnabled = $state(true);
	let ga4Enabled = $state(false);
	let ga4Id = $state('');
	let umamiEnabled = $state(false);
	let umamiUrl = $state('');
	let umamiSiteId = $state('');
	let crmWebhookUrl = $state('');
	let crmWebhookSecret = $state('');
	let cookieBannerSnippet = $state('');
	let personalizationEnabled = $state(false);
	let identityMaxAgeDays = $state('30');

	type State = {
		atomicsiteTrackingEnabled: boolean;
		ga4Enabled: boolean;
		ga4Id: string;
		umamiEnabled: boolean;
		umamiUrl: string;
		umamiSiteId: string;
		crmWebhookUrl: string;
		crmWebhookSecret: string;
		cookieBannerSnippet: string;
		personalizationEnabled: boolean;
		identityMaxAgeDays: string;
	};

	let initial: State = $state({
		atomicsiteTrackingEnabled: true,
		ga4Enabled: false,
		ga4Id: '',
		umamiEnabled: false,
		umamiUrl: '',
		umamiSiteId: '',
		crmWebhookUrl: '',
		crmWebhookSecret: '',
		cookieBannerSnippet: '',
		personalizationEnabled: false,
		identityMaxAgeDays: '30'
	});

	async function load() {
		loading = true;
		try {
			const rows = await settingsApi.listByCategory(siteID, 'analytics');
			const m = categoryMap(rows);

			atomicsiteTrackingEnabled =
				m.atomicsite_tracking_enabled === undefined
					? true
					: toBool(m.atomicsite_tracking_enabled);
			ga4Id = m.ga4_id || '';
			ga4Enabled = ga4Id.length > 0 || toBool(m.ga4_enabled);
			umamiUrl = m.umami_url || '';
			umamiSiteId = m.umami_site_id || '';
			umamiEnabled = (umamiUrl.length > 0 && umamiSiteId.length > 0) || toBool(m.umami_enabled);
			crmWebhookUrl = m.crm_webhook_url || '';
			crmWebhookSecret = m.crm_webhook_secret || '';
			cookieBannerSnippet = m.cookie_banner_snippet || '';
			personalizationEnabled = toBool(m.personalization_enabled);
			identityMaxAgeDays = m.identity_max_age_days || '30';

			initial = {
				atomicsiteTrackingEnabled,
				ga4Enabled,
				ga4Id,
				umamiEnabled,
				umamiUrl,
				umamiSiteId,
				crmWebhookUrl,
				crmWebhookSecret,
				cookieBannerSnippet,
				personalizationEnabled,
				identityMaxAgeDays
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
		atomicsiteTrackingEnabled !== initial.atomicsiteTrackingEnabled ||
			ga4Enabled !== initial.ga4Enabled ||
			ga4Id !== initial.ga4Id ||
			umamiEnabled !== initial.umamiEnabled ||
			umamiUrl !== initial.umamiUrl ||
			umamiSiteId !== initial.umamiSiteId ||
			crmWebhookUrl !== initial.crmWebhookUrl ||
			crmWebhookSecret !== initial.crmWebhookSecret ||
			cookieBannerSnippet !== initial.cookieBannerSnippet ||
			personalizationEnabled !== initial.personalizationEnabled ||
			identityMaxAgeDays !== initial.identityMaxAgeDays
	);

	function discard() {
		atomicsiteTrackingEnabled = initial.atomicsiteTrackingEnabled;
		ga4Enabled = initial.ga4Enabled;
		ga4Id = initial.ga4Id;
		umamiEnabled = initial.umamiEnabled;
		umamiUrl = initial.umamiUrl;
		umamiSiteId = initial.umamiSiteId;
		crmWebhookUrl = initial.crmWebhookUrl;
		crmWebhookSecret = initial.crmWebhookSecret;
		cookieBannerSnippet = initial.cookieBannerSnippet;
		personalizationEnabled = initial.personalizationEnabled;
		identityMaxAgeDays = initial.identityMaxAgeDays;
	}

	function b(v: boolean): string {
		return v ? '1' : '0';
	}

	async function save() {
		if (saving || !dirty) return;
		saving = true;
		try {
			const items: settingsApi.SettingUpsertInput[] = [
				{
					category: 'analytics',
					key: 'atomicsite_tracking_enabled',
					value: b(atomicsiteTrackingEnabled)
				},
				{ category: 'analytics', key: 'ga4_enabled', value: b(ga4Enabled) },
				{ category: 'analytics', key: 'ga4_id', value: ga4Enabled ? ga4Id : '' },
				{ category: 'analytics', key: 'umami_enabled', value: b(umamiEnabled) },
				{ category: 'analytics', key: 'umami_url', value: umamiEnabled ? umamiUrl : '' },
				{
					category: 'analytics',
					key: 'umami_site_id',
					value: umamiEnabled ? umamiSiteId : ''
				},
				{ category: 'analytics', key: 'crm_webhook_url', value: crmWebhookUrl },
				{ category: 'analytics', key: 'crm_webhook_secret', value: crmWebhookSecret },
				{ category: 'analytics', key: 'cookie_banner_snippet', value: cookieBannerSnippet },
				{
					category: 'analytics',
					key: 'personalization_enabled',
					value: b(personalizationEnabled)
				},
				{
					category: 'analytics',
					key: 'identity_max_age_days',
					value: identityMaxAgeDays
				}
			];
			await settingsApi.bulkUpsert(siteID, items);

			initial = {
				atomicsiteTrackingEnabled,
				ga4Enabled,
				ga4Id,
				umamiEnabled,
				umamiUrl,
				umamiSiteId,
				crmWebhookUrl,
				crmWebhookSecret,
				cookieBannerSnippet,
				personalizationEnabled,
				identityMaxAgeDays
			};
			toast.success('Analytics settings saved.');
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
			Analytics
		</h1>
		<p class="text-[13px] text-text-secondary">
			Tracking, consent and CRM hooks. All optional, all GDPR-aware.
		</p>
	</header>

	{#if loading}
		<div class="mt-8 flex items-center justify-center py-12">
			<Spinner />
		</div>
	{:else}
		<div class="mt-8 flex flex-col gap-5">
			<Card padding="md">
				<div class="flex gap-3">
					<div
						class="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-accent/10 text-accent"
					>
						<svg
							class="h-3.5 w-3.5"
							fill="none"
							viewBox="0 0 24 24"
							stroke-width="2"
							stroke="currentColor"
						>
							<path
								stroke-linecap="round"
								stroke-linejoin="round"
								d="M11.25 11.25l.041-.02a.75.75 0 011.063.852l-.708 2.836a.75.75 0 001.063.853l.041-.021M21 12a9 9 0 11-18 0 9 9 0 0118 0zm-9-3.75h.008v.008H12V8.25z"
							/>
						</svg>
					</div>
					<div class="flex-1">
						<p class="text-[13px] text-text-primary">
							Analytics start collecting once a site is deployed and the next build runs.
						</p>
						<p class="mt-1 text-[12px] text-text-muted">
							Server-side tracking via Nginx is immune to ad blockers. Identification only happens
							after the visitor accepts marketing consent.
						</p>
					</div>
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Atomicsite tracking
				</h2>
				<div class="mt-4 flex items-center justify-between gap-4">
					<div class="flex flex-col">
						<span class="text-[13px] text-text-primary">Server-side visit tracking</span>
						<span class="text-[12px] text-text-muted">
							Logs page views from Nginx access logs. No client-side script.
						</span>
					</div>
					<Switch bind:checked={atomicsiteTrackingEnabled} ariaLabel="Enable Atomicsite tracking" />
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Custom cookie banner snippet
				</h2>
				<p class="mt-2 text-[12px] text-text-muted">
					Bring your own banner: Cookiebot, OneTrust, Termly, Iubenda, anything. Paste raw HTML
					or a script tag. We inject it verbatim into &lt;head&gt; on every page. Leave empty if
					you use CookieProof or no banner.
				</p>
				<div class="mt-3">
					<Textarea
						rows={6}
						placeholder={`<script src="https://consent.cookiebot.com/uc.js" data-cbid="..." async></script>`}
						bind:value={cookieBannerSnippet}
					/>
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Cookie consent banner
				</h2>
				<p class="mt-2 text-[12px] text-text-muted">
					Atomic Site ships a same-origin cookie banner that auto-styles to your brand and
					stores consent proofs in your own database. Configure it on the dedicated
					<a class="underline" href={`/sites/${siteID}/cookies`}>Cookies page</a>.
				</p>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Google Analytics 4
				</h2>
				<div class="mt-4 flex flex-col gap-3">
					<div class="flex items-center justify-between gap-4">
						<div class="flex flex-col">
							<span class="text-[13px] text-text-primary">Enable GA4</span>
							<span class="text-[12px] text-text-muted">
								Loaded only after consent if CookieProof is on.
							</span>
						</div>
						<Switch bind:checked={ga4Enabled} ariaLabel="Enable GA4" />
					</div>
					{#if ga4Enabled}
						<Input
							label="Measurement ID"
							placeholder="G-XXXXXXXXXX"
							bind:value={ga4Id}
						/>
					{/if}
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Umami</h2>
				<div class="mt-4 flex flex-col gap-3">
					<div class="flex items-center justify-between gap-4">
						<div class="flex flex-col">
							<span class="text-[13px] text-text-primary">Enable Umami</span>
							<span class="text-[12px] text-text-muted">
								Cookieless privacy-friendly analytics.
							</span>
						</div>
						<Switch bind:checked={umamiEnabled} ariaLabel="Enable Umami" />
					</div>
					{#if umamiEnabled}
						<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
							<Input
								label="Umami URL"
								placeholder="https://analytics.example.com"
								bind:value={umamiUrl}
							/>
							<Input
								label="Site ID"
								placeholder="00000000-0000-0000-0000-000000000000"
								bind:value={umamiSiteId}
							/>
						</div>
					{/if}
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					CRM webhook
				</h2>
				<div class="mt-4 flex flex-col gap-3">
					<Input
						label="Webhook URL"
						placeholder="https://crm.example.com/api/leads"
						hint="Form submissions and identification events on this site POST here."
						bind:value={crmWebhookUrl}
					/>
					<Input
						label="Webhook secret"
						type="password"
						placeholder="shared signing secret"
						hint="Sent as X-Atomicsite-Signature for HMAC verification."
						bind:value={crmWebhookSecret}
					/>
				</div>
			</Card>

			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Personalization (CRM round-trip)
				</h2>
				<p class="mt-2 text-[12px] text-text-muted">
					Bidirectional loop: built sites read per-visitor metadata your CRM pushes back, and
					blocks gated on a tiny DSL ("lead_score &gt;= 60", "name present") reveal on match.
					See
					<a href="/docs/personalization" class="text-text-primary hover:underline">
						Documentation -&gt; Personalization
					</a>
					for the full model.
				</p>
				<div class="mt-4 flex flex-col gap-4">
					<div class="flex items-center justify-between gap-4">
						<div class="flex flex-col">
							<span class="text-[13px] text-text-primary">
								Enable visitor hydration script
							</span>
							<span class="text-[12px] text-text-muted">
								Inject a small &lt;script&gt; into every built page that fetches /t/visitor
								and reveals matching conditional blocks. Off by default.
							</span>
						</div>
						<Switch
							bind:checked={personalizationEnabled}
							ariaLabel="Enable visitor personalization hydration"
						/>
					</div>
					{#if personalizationEnabled}
						<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
							<Input
								label="Identity max age (days)"
								type="number"
								placeholder="30"
								hint="Identified-tier fields (name, email) are scrubbed if the CRM's identity_confirmed_at is older than this. Default 30 days."
								bind:value={identityMaxAgeDays}
							/>
							<div class="flex flex-col gap-1.5">
								<span class="text-[12px] font-medium text-text-secondary">Status</span>
								<div
									class="rounded-lg border border-border-light bg-bg-elevated px-3 py-2 text-[12px] text-text-secondary"
								>
									{#if !crmWebhookUrl}
										<span class="text-amber-600 dark:text-amber-400">
											Wire a CRM webhook above to receive metadata pushes back.
										</span>
									{:else}
										<span class="text-emerald-700 dark:text-emerald-400">
											CRM webhook configured. Metadata pushes will land at /t/inbound on
											the next event.
										</span>
									{/if}
								</div>
							</div>
						</div>
						<div class="rounded-lg border border-border-light bg-bg-elevated px-3 py-2 text-[11px] text-text-muted">
							The hydration script ships on the next build. Trigger one from the Build page
							after saving.
						</div>
					{/if}
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

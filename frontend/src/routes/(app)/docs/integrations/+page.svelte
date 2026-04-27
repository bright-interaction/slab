<script lang="ts">
	import Card from '$lib/components/ui/Card.svelte';
	import { ArrowLeft } from 'lucide-svelte';
	import { currentSite } from '$lib/stores/currentSite.svelte';

	const siteID = $derived(currentSite.value?.id ?? null);

	type Status = 'shipped' | 'planned';
	type Integration = {
		title: string;
		status: Status;
		body: string;
		linkLabel: string;
		linkHref: string | null;
	};

	const integrations = $derived.by<Integration[]>(() => {
		const settings = (path: string) => (siteID ? `/sites/${siteID}/settings/${path}` : null);
		const site = (path: string) => (siteID ? `/sites/${siteID}/${path}` : null);
		return [
			{
				title: 'Bring your own CRM',
				status: 'shipped',
				body:
					'Paste any HTTPS webhook URL plus a shared secret. Form submissions, identification events, and consent updates POST to your endpoint, signed with HMAC-SHA256 in the X-Atomicsite-Signature header. Works with HubSpot, Salesforce, Pipedrive, BrightCRM, n8n, Zapier, your own server, anything that speaks HTTP.',
				linkLabel: 'Settings → Analytics → CRM webhook',
				linkHref: settings('analytics')
			},
			{
				title: 'Bring your own consent banner',
				status: 'shipped',
				body:
					'Paste raw HTML or a script tag for any cookie banner: Cookiebot, OneTrust, Termly, Iubenda, your own. We inject it verbatim into <head> on every page. Use this if you already have a consent platform; reach for CookieProof if you want one bundled.',
				linkLabel: 'Settings → Analytics → Custom cookie banner snippet',
				linkHref: settings('analytics')
			},
			{
				title: 'CookieProof',
				status: 'shipped',
				body:
					'One-click GDPR consent banner. Snippet auto-injected, /t/consent receives consent:update + consent:gpc events, the engagement beacon waits for consent:init with analytics:true before firing. Right for users who want consent + multi-domain + audit logs without wiring a third party.',
				linkLabel: 'Settings → Analytics → CookieProof consent',
				linkHref: settings('analytics')
			},
			{
				title: 'BrightCRM',
				status: 'shipped',
				body:
					'A specific preset on top of the generic CRM webhook. Same HMAC contract, but the receiver maps Atomic Site activity types onto the BrightCRM enum so anonymous and identified journeys stitch together inside the CRM timeline.',
				linkLabel: 'Settings → Analytics → CRM webhook',
				linkHref: settings('analytics')
			},
			{
				title: 'Google Analytics 4',
				status: 'shipped',
				body:
					'Drop-in GA4 with a Measurement ID. Loaded only after consent if a banner is in place (CookieProof or your custom snippet that sets the GCM denied/granted state).',
				linkLabel: 'Settings → Analytics → Google Analytics 4',
				linkHref: settings('analytics')
			},
			{
				title: 'Umami',
				status: 'shipped',
				body:
					'Cookieless privacy-friendly analytics. Point at any Umami host plus a site ID. Right for users who want a non-Google, GDPR-clean alternative.',
				linkLabel: 'Settings → Analytics → Umami',
				linkHref: settings('analytics')
			},
			{
				title: 'Dockyard',
				status: 'shipped',
				body:
					'Deploy target. Requires https:// URL plus a UUID server_id. Posts to /api/servers/{id}/proxy/routes then /proxy/apply. Right for built sites that should land on a custom domain instead of the wildcard.',
				linkLabel: 'Settings → Deployment',
				linkHref: settings('deployment')
			},
			{
				title: 'Figma',
				status: 'shipped',
				body:
					'Design tokens import. POST /api/sites/{id}/figma/import pulls published FILL and TEXT styles, slugifies names ("Brand / Primary 600" -> .color-brand-primary-600), seeds CSS classes, and updates site.primary_color + fonts when current values are wizard defaults. Token never persisted.',
				linkLabel: 'Branding (Design references area)',
				linkHref: site('branding')
			},
			{
				title: 'GitHub design references',
				status: 'shipped',
				body:
					'Read-only pattern reference. Public repos only. Fetches package.json, README, tailwind config, common globals.css, and 5 representative components. 32 KB per file, 200 KB total cap. Bundle flows into /api/agent/context as design_references[] so the agent reads the design vocabulary.',
				linkLabel: 'Branding',
				linkHref: site('branding')
			},
			{
				title: 'Cloudflare DNS',
				status: 'shipped',
				body:
					'Wildcard record *.atomicsite.example.com -> 203.0.113.10 covers any built-site slug. Wildcard TLS cert covers the same. No per-site DNS or cert work needed for slugs on the default domain.',
				linkLabel: 'Architecture page',
				linkHref: '/docs/architecture'
			},
			{
				title: 'Self-hosted woff2 fonts',
				status: 'shipped',
				body:
					'POST /api/sites/{id}/fonts. Magic header (wOF2 at offset 0) validated; woff/ttf/otf rejected. 2 MB cap per file. Served at /atomicsite-fonts/{site_id}/{font_id}.woff2 with Cache-Control: immutable + crossorigin. Builder emits @font-face + rel="preload" hints.',
				linkLabel: 'Branding',
				linkHref: site('branding')
			},
			{
				title: 'Site Inspector evaluation engine',
				status: 'shipped',
				body:
					'130+ checks ported from the Chrome extension. Run after every build. 13-grade scale, A+ through F. Results expose to the agent through GET /api/agent/evaluation/{buildID} so it can self-correct.',
				linkLabel: 'Build',
				linkHref: site('build')
			},
			{
				title: 'Sentinel',
				status: 'planned',
				body:
					'Uptime + GRC monitoring for built sites. Will hook into the deploy pipeline so a site goes into a watchlist as soon as it ships.',
				linkLabel: null as never,
				linkHref: null
			},
			{
				title: 'SVAR',
				status: 'planned',
				body:
					'Security scan integration. Will run a full SVAR scan on each build and surface findings in the evaluation panel alongside the in-house security checks.',
				linkLabel: null as never,
				linkHref: null
			}
		];
	});
</script>

<div class="mx-auto max-w-5xl px-6 py-8">
	<a
		href="/docs"
		class="inline-flex items-center gap-1.5 text-[12px] text-text-muted transition-colors hover:text-text-primary"
	>
		<ArrowLeft size={14} strokeWidth={1.75} />
		Documentation
	</a>
	<header class="mt-3 flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
			Integrations
		</h1>
		<p class="text-[13px] text-text-secondary">
			External systems Atomic Site connects to, what they do, and where to wire them up.
		</p>
	</header>

	<div class="mt-8 grid grid-cols-1 gap-3 md:grid-cols-2">
		{#each integrations as it (it.title)}
			<Card padding="md">
				<div class="flex items-start justify-between gap-3">
					<h2 class="text-[14px] font-medium text-text-primary">{it.title}</h2>
					<span
						class="inline-flex items-center rounded-full px-2 py-0.5 font-mono text-[10px] {it.status ===
						'shipped'
							? 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'
							: 'bg-amber-500/10 text-amber-600 dark:text-amber-400'}"
					>
						{it.status}
					</span>
				</div>
				<p class="mt-2 text-[12px] text-text-secondary">{it.body}</p>
				{#if it.linkHref}
					<a
						href={it.linkHref}
						class="mt-3 inline-flex items-center gap-1.5 text-[12px] text-text-primary transition-colors hover:text-text-secondary"
					>
						{it.linkLabel}
					</a>
				{/if}
			</Card>
		{/each}
	</div>
</div>

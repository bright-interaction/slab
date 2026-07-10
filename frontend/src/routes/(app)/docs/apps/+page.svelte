<script lang="ts">
	import Card from '$lib/components/ui/Card.svelte';
	import { ArrowLeft } from 'lucide-svelte';
	import { currentSite } from '$lib/stores/currentSite.svelte';

	const siteID = $derived(currentSite.value?.id ?? null);

	type Status = 'shipped' | 'preset' | 'byo' | 'planned';
	type App = {
		name: string;
		status: Status;
		body: string;
		setup: string;
		linkLabel: string | null;
		linkHref: string | null;
	};
	type Section = {
		title: string;
		intro: string;
		apps: App[];
	};

	const sections = $derived.by<Section[]>(() => {
		const settings = (path: string) =>
			siteID ? `/sites/${siteID}/settings/${path}` : '/sites';
		const site = (path: string) => (siteID ? `/sites/${siteID}/${path}` : '/sites');
		return [
			{
				title: 'Bright Interaction stack',
				intro: 'Built-in connectors to our own apps. Toggle on, paste an ID, ship.',
				apps: [
					{
						name: 'BrightCRM',
						status: 'preset',
						body: 'Sync visitor events into the BrightCRM timeline. HMAC-signed, activity types map onto the BrightCRM enum.',
						setup: 'Settings -> Analytics -> CRM webhook. Paste your BrightCRM webhook URL and shared secret.',
						linkLabel: 'Settings -> Analytics',
						linkHref: settings('analytics')
					},
					{
						name: 'CookieProof',
						status: 'shipped',
						body: 'GDPR consent banner with built-in /t/consent receiver, GCM defaults, and audit-friendly logs.',
						setup: 'Settings -> Analytics -> CookieProof. Toggle on and paste your organisation ID.',
						linkLabel: 'Settings -> Analytics',
						linkHref: settings('analytics')
					},
					{
						name: 'Dockyard',
						status: 'shipped',
						body: 'Self-hosted container platform. Deploy built sites onto a custom domain via Caddy/Traefik routes.',
						setup: 'Settings -> Deployment -> Add a Dockyard target with the server URL and UUID server_id.',
						linkLabel: 'Settings -> Deployment',
						linkHref: settings('deployment')
					},
					{
						name: 'Sentinel',
						status: 'planned',
						body: 'Uptime + GRC. Built sites will auto-register on the deploy hook so they land in the watchlist immediately.',
						setup: 'Coming soon. Will live next to Deployment in Settings.',
						linkLabel: null,
						linkHref: null
					},
					{
						name: 'SVAR',
						status: 'planned',
						body: 'Full security scan on every build. Findings will appear in the evaluation panel alongside the in-house security checks.',
						setup: 'Coming soon. Will run as part of the post-build pipeline.',
						linkLabel: null,
						linkHref: null
					}
				]
			},
			{
				title: 'CRM connectors',
				intro:
					'Bring your own CRM via the generic webhook. Form submissions, identification events, and consent updates POST to your endpoint, signed with HMAC-SHA256 in the X-Atomicsite-Signature header.',
				apps: [
					{
						name: 'HubSpot',
						status: 'byo',
						body: 'Drop a HubSpot Workflow with a webhook trigger. Map our payload fields (visitor_id, email, page) to HubSpot contact properties.',
						setup: 'Workflow URL -> Settings -> Analytics -> CRM webhook URL. Use any token as the secret; HubSpot ignores it but our HMAC still works.',
						linkLabel: 'Settings -> Analytics',
						linkHref: settings('analytics')
					},
					{
						name: 'Salesforce',
						status: 'byo',
						body: 'Use Web-to-Lead, a Flow with HTTP Callout, or a custom Apex REST endpoint. Verify HMAC on the receiver side for security.',
						setup: 'Endpoint URL + shared secret -> Settings -> Analytics -> CRM webhook.',
						linkLabel: 'Settings -> Analytics',
						linkHref: settings('analytics')
					},
					{
						name: 'Pipedrive',
						status: 'byo',
						body: 'Pipedrive Automations -> Webhook trigger. Maps cleanly to Person + Deal endpoints downstream.',
						setup: 'Automation webhook URL -> Settings -> Analytics -> CRM webhook.',
						linkLabel: 'Settings -> Analytics',
						linkHref: settings('analytics')
					},
					{
						name: 'Attio / Folk / Close / any other CRM',
						status: 'byo',
						body: 'Any CRM that accepts an inbound HTTPS webhook works. We do the heavy lifting (HMAC, retries, throttle, payload schema).',
						setup: 'Settings -> Analytics -> CRM webhook URL + secret. Done.',
						linkLabel: 'Settings -> Analytics',
						linkHref: settings('analytics')
					}
				]
			},
			{
				title: 'Analytics',
				intro: 'Drop-in scripts. Loaded only after consent if a banner is in place.',
				apps: [
					{
						name: 'Google Analytics 4',
						status: 'shipped',
						body: 'GA4 measurement script with a Measurement ID. Loads after CookieProof grants analytics consent (or your custom snippet sets the GCM granted state).',
						setup: 'Settings -> Analytics -> Google Analytics 4. Paste G-XXXXXXXXXX.',
						linkLabel: 'Settings -> Analytics',
						linkHref: settings('analytics')
					},
					{
						name: 'Umami',
						status: 'shipped',
						body: 'Cookieless privacy-friendly analytics. Self-host or use Umami Cloud. No banner gating needed.',
						setup: 'Settings -> Analytics -> Umami. Paste host URL + Site ID.',
						linkLabel: 'Settings -> Analytics',
						linkHref: settings('analytics')
					},
					{
						name: 'Google Tag Manager / Plausible / Matomo / Fathom',
						status: 'byo',
						body: 'Paste their script tag into Allowed scripts (CSP whitelist) plus a snippet in the Cookie banner field if needed.',
						setup: 'Settings -> Allowed scripts and Settings -> Analytics -> Cookie banner snippet.',
						linkLabel: 'Settings -> Allowed scripts',
						linkHref: settings('allowed-scripts')
					}
				]
			},
			{
				title: 'Cookie consent',
				intro: 'CookieProof is bundled. Or paste any other banner snippet verbatim into <head>.',
				apps: [
					{
						name: 'Cookiebot',
						status: 'byo',
						body: 'Pastes a single <script> tag from your Cookiebot domain group. Sets the GCM defaults so GA4 + Ads behave correctly.',
						setup: 'Settings -> Analytics -> Custom cookie banner snippet.',
						linkLabel: 'Settings -> Analytics',
						linkHref: settings('analytics')
					},
					{
						name: 'OneTrust',
						status: 'byo',
						body: 'Enterprise consent. Paste the OneTrust autoBlock + sdk script tags.',
						setup: 'Settings -> Analytics -> Custom cookie banner snippet.',
						linkLabel: 'Settings -> Analytics',
						linkHref: settings('analytics')
					},
					{
						name: 'Termly',
						status: 'byo',
						body: 'GDPR + CCPA + LGPD consent. Single script tag.',
						setup: 'Settings -> Analytics -> Custom cookie banner snippet.',
						linkLabel: 'Settings -> Analytics',
						linkHref: settings('analytics')
					},
					{
						name: 'Iubenda',
						status: 'byo',
						body: 'EU-focused consent + privacy policy generator.',
						setup: 'Settings -> Analytics -> Custom cookie banner snippet.',
						linkLabel: 'Settings -> Analytics',
						linkHref: settings('analytics')
					}
				]
			},
			{
				title: 'Automation platforms',
				intro:
					'Anything that speaks HTTP can call the agent API or receive our webhooks. The agent API needs an X-Agent-Key header; the CRM webhook receiver needs to verify HMAC.',
				apps: [
					{
						name: 'n8n',
						status: 'byo',
						body: 'Webhook trigger node receives our events; HTTP Request node calls the agent API for scheduled rebuilds, content sync from a CMS, pre-flight evaluation runs.',
						setup: 'Outbound: paste the n8n webhook URL into Settings -> Analytics -> CRM webhook. Inbound: HTTP Request node + X-Agent-Key header.',
						linkLabel: 'Connect your agent (n8n example)',
						linkHref: '/docs/connect-your-agent'
					},
					{
						name: 'Make (Integromat)',
						status: 'byo',
						body: 'Same model: Webhook module receives, HTTP module calls.',
						setup: 'Outbound: paste Make webhook URL. Inbound: HTTP module with X-Agent-Key header.',
						linkLabel: 'Connect your agent',
						linkHref: '/docs/connect-your-agent'
					},
					{
						name: 'Zapier',
						status: 'byo',
						body: 'Webhooks by Zapier (Catch + Custom Request) covers both directions.',
						setup: 'Outbound: paste Zapier Catch URL. Inbound: Custom Request action with X-Agent-Key header.',
						linkLabel: 'Connect your agent',
						linkHref: '/docs/connect-your-agent'
					}
				]
			},
			{
				title: 'AI agent runtimes',
				intro:
					'Anything that can run curl can build sites. The bundle on /docs/connect-your-agent gives you a personalised CLAUDE.md with the key, base URL, and pending_setup snapshot baked in.',
				apps: [
					{
						name: 'Claude Code / Claude CLI',
						status: 'shipped',
						body: 'One-click setup. Generate the bundle, drop CLAUDE.md + atomicsite.env at your project root, run claude. The agent walks pending_setup with you, then builds.',
						setup: 'Documentation -> Connect your agent -> Generate key & download CLAUDE.md.',
						linkLabel: 'Connect your agent',
						linkHref: '/docs/connect-your-agent'
					},
					{
						name: 'Cursor / VS Code agents',
						status: 'byo',
						body: 'Same bundle. Rename CLAUDE.md to .cursorrules and feed atomicsite.env into the agent runtime.',
						setup: 'Documentation -> Connect your agent.',
						linkLabel: 'Connect your agent',
						linkHref: '/docs/connect-your-agent'
					},
					{
						name: 'CI / GitHub Actions',
						status: 'byo',
						body: 'Store the agent key as a repo secret, run a curl-based job on push to trigger a build and gate the merge on the evaluation grade.',
						setup: 'Documentation -> Connect your agent (CI section).',
						linkLabel: 'Connect your agent',
						linkHref: '/docs/connect-your-agent'
					}
				]
			},
			{
				title: 'Notifications',
				intro: 'Build pass/fail and evaluation regressions out to your team chat.',
				apps: [
					{
						name: 'Slack',
						status: 'planned',
						body: 'Incoming webhook integration. We will POST build events with the eval grade per category.',
						setup: 'Coming soon. Today: receive the CRM webhook into n8n / Zapier and forward to Slack.',
						linkLabel: null,
						linkHref: null
					},
					{
						name: 'Discord / Microsoft Teams',
						status: 'planned',
						body: 'Same webhook model.',
						setup: 'Coming soon.',
						linkLabel: null,
						linkHref: null
					}
				]
			},
			{
				title: 'Email + transactional',
				intro: 'Form submissions, contact enquiries, lifecycle emails.',
				apps: [
					{
						name: 'Resend / Postmark / SendGrid',
						status: 'planned',
						body: 'API key per provider. Forms route through the platform; bounce + open events fold into analytics.',
						setup: 'Coming soon.',
						linkLabel: null,
						linkHref: null
					},
					{
						name: 'Mailchimp / Brevo / Klaviyo / ConvertKit',
						status: 'byo',
						body: 'Today: wire form submissions into n8n / Zapier and forward to your ESP API.',
						setup: 'Use the CRM webhook as a generic outbound channel.',
						linkLabel: 'Settings -> Analytics',
						linkHref: settings('analytics')
					}
				]
			},
			{
				title: 'E-commerce',
				intro: 'For the ecommerce starter kit + product blocks.',
				apps: [
					{
						name: 'Stripe',
						status: 'shipped',
						body: 'Stripe Checkout redirect baked into the ecommerce starter kit. Product/Offer microdata emitted automatically.',
						setup: 'Wizard -> pick the ecommerce kit. Set publishable + secret keys in Settings.',
						linkLabel: 'Settings',
						linkHref: site('settings')
					},
					{
						name: 'Shopify / WooCommerce',
						status: 'planned',
						body: 'Pull product catalogue + inventory; push order events to the CRM webhook.',
						setup: 'Coming soon.',
						linkLabel: null,
						linkHref: null
					}
				]
			},
			{
				title: 'Design',
				intro: 'Pull design vocabulary into the agent context.',
				apps: [
					{
						name: 'Figma',
						status: 'shipped',
						body: 'Token import. Pulls FILL + TEXT styles, slugifies names, seeds CSS classes, optionally updates primary_color + fonts. Token never persisted.',
						setup: 'Branding -> Design references area -> Figma import.',
						linkLabel: 'Branding',
						linkHref: site('branding')
					},
					{
						name: 'GitHub design references',
						status: 'shipped',
						body: 'Paste a public repo URL. We fetch package.json, README, tailwind config, globals.css, and 5 representative components. Bundle flows into /api/agent/context as design_references[].',
						setup: 'Branding -> Design references area -> Add reference.',
						linkLabel: 'Branding',
						linkHref: site('branding')
					}
				]
			},
			{
				title: 'Search Console + AI search',
				intro: 'Verify ownership and feed AI crawlers.',
				apps: [
					{
						name: 'Google Search Console / Bing Webmaster',
						status: 'byo',
						body: 'Add the verification meta tag via the SEO settings or a global block.',
						setup: 'Settings -> SEO. Or a meta verification block on the homepage.',
						linkLabel: 'Settings -> SEO',
						linkHref: settings('seo')
					},
					{
						name: 'AI crawlers (GPTBot, ClaudeBot, PerplexityBot)',
						status: 'shipped',
						body: 'llms.txt, robots.txt, and Organization JSON-LD all emit by default. AI search compatibility is graded by the GEO eval category.',
						setup: 'Automatic. Inspect at Build -> latest evaluation -> SEO/GEO.',
						linkLabel: 'Build',
						linkHref: site('build')
					}
				]
			},
			{
				title: 'Platform',
				intro:
					'Platform-level mechanics that run under the hood. Not user-toggleable; documented so the agent and the human admin know what is in play.',
				apps: [
					{
						name: 'DNS provider (any)',
						status: 'shipped',
						body:
							'For multi-tenant deployments, point a wildcard record (e.g. *.tenants.example.com) at the host running atomicsite. Wildcard TLS works the same way. Single-root deployments only need an A/AAAA record for ATOMICSITE_PRIMARY_DOMAIN. Custom domains route via the deploy target you configure (Dockyard, Fly, Vercel, plain VPS).',
						setup: 'Automatic. Inspect at /docs/architecture for the topology.',
						linkLabel: 'Architecture',
						linkHref: '/docs/architecture'
					},
					{
						name: 'Self-hosted woff2 fonts',
						status: 'shipped',
						body:
							'POST /api/sites/{id}/fonts. Magic header (wOF2 at offset 0) validated; woff/ttf/otf rejected. 2 MB cap per file. Served at /atomicsite-fonts/{site_id}/{font_id}.woff2 with Cache-Control: immutable + crossorigin. Builder emits @font-face + rel="preload" hints.',
						setup: 'Branding -> Custom fonts.',
						linkLabel: 'Branding',
						linkHref: site('branding')
					},
					{
						name: 'Site Inspector evaluation engine',
						status: 'shipped',
						body:
							'130+ checks ported from the Chrome extension. Run after every build across Security, SEO, GEO, Performance, Accessibility, Privacy. 13-grade scale, A+ through F. Results expose to the agent through GET /api/agent/evaluation/{buildID} so it can self-correct.',
						setup: 'Automatic. Inspect at Build -> latest evaluation.',
						linkLabel: 'Build',
						linkHref: site('build')
					},
					{
						name: 'Wildcard static host (Caddy / nginx)',
						status: 'shipped',
						body:
							'For multi-tenant deployments, a wildcard route on Caddy or nginx serves built sites straight from a dist/ directory on disk (e.g. /srv/atomicsite/<slug>/dist). Local-kind deploy targets land here. Single-root deployments push the dist/ output to any static host instead.',
						setup: 'Operator-configured. See /docs/architecture for an example Caddy block.',
						linkLabel: null,
						linkHref: null
					},
					{
						name: 'CI deploy pipeline (any)',
						status: 'shipped',
						body:
							'Push to main triggers your CI workflow: Bun build, Docker build with frontend embedded via go:embed, container push, container restart. End-to-end ~2-3 minutes from push to live. Reference workflow lives at .github/workflows/oss-ci.yml; adapt it to whichever CI you run.',
						setup: 'Operator-configured. GitHub Actions, GitLab CI, and other CI systems all work.',
						linkLabel: null,
						linkHref: null
					},
					{
						name: 'Sentinel',
						status: 'planned',
						body:
							'Uptime + GRC monitoring for built sites. Will hook into the deploy pipeline so a site goes into a watchlist as soon as it ships.',
						setup: 'Coming soon.',
						linkLabel: null,
						linkHref: null
					},
					{
						name: 'SVAR',
						status: 'planned',
						body:
							'Security scan integration. Will run a full SVAR scan on each build and surface findings in the evaluation panel alongside the in-house security checks.',
						setup: 'Coming soon.',
						linkLabel: null,
						linkHref: null
					}
				]
			}
		];
	});

	const statusLabel: Record<Status, string> = {
		shipped: 'shipped',
		preset: 'shipped (preset)',
		byo: 'bring your own',
		planned: 'planned'
	};
	const statusClass: Record<Status, string> = {
		shipped: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
		preset: 'bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
		byo: 'bg-sky-500/10 text-sky-600 dark:text-sky-400',
		planned: 'bg-amber-500/10 text-amber-600 dark:text-amber-400'
	};
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
			Apps & integrations
		</h1>
		<p class="text-[13px] text-text-secondary">
			Easy connectors to our own apps and to third parties. Most are wired with a single API
			key or webhook URL on Settings -> Analytics. The
			<a href="#platform" class="text-text-primary hover:underline">Platform</a> section at the bottom
			documents platform-level mechanics that run under the hood.
		</p>
	</header>

	<div class="mt-6 flex flex-wrap items-center gap-3 text-[11px] text-text-muted">
		<span class="inline-flex items-center gap-1.5">
			<span class="h-2 w-2 rounded-full bg-emerald-500"></span>
			shipped
		</span>
		<span class="inline-flex items-center gap-1.5">
			<span class="h-2 w-2 rounded-full bg-sky-500"></span>
			bring your own (works today, no first-party UI yet)
		</span>
		<span class="inline-flex items-center gap-1.5">
			<span class="h-2 w-2 rounded-full bg-amber-500"></span>
			planned
		</span>
	</div>

	<div class="mt-6 flex flex-col gap-6">
		{#each sections as section (section.title)}
			<div id={section.title === 'Platform' ? 'platform' : undefined}>
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					{section.title}
				</h2>
				<p class="mt-2 text-[13px] text-text-secondary">{section.intro}</p>
				<div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2">
					{#each section.apps as app (app.name)}
						<Card padding="md">
							<div class="flex items-start justify-between gap-3">
								<h3 class="text-[14px] font-medium text-text-primary">{app.name}</h3>
								<span
									class="inline-flex items-center rounded-full px-2 py-0.5 font-mono text-[10px] {statusClass[
										app.status
									]}"
								>
									{statusLabel[app.status]}
								</span>
							</div>
							<p class="mt-2 text-[12px] text-text-secondary">{app.body}</p>
							<p class="mt-2 text-[11px] text-text-muted">
								<strong class="font-medium text-text-secondary">Setup:</strong>
								{app.setup}
							</p>
							{#if app.linkHref}
								<a
									href={app.linkHref}
									class="mt-3 inline-flex items-center gap-1.5 text-[12px] text-text-primary transition-colors hover:text-text-secondary"
								>
									{app.linkLabel}
								</a>
							{/if}
						</Card>
					{/each}
				</div>
			</div>
		{/each}
	</div>

	<Card padding="md" class="mt-6">
		<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
			Don't see your app?
		</h2>
		<p class="mt-3 text-[13px] text-text-secondary">
			Anything that accepts an HTTPS webhook works as a CRM/event sink. Anything with a script
			tag works as a banner or analytics tracker. Anything with an HTTP API can be called from
			your agent or from n8n. The platform stays out of the way; you wire the rest.
		</p>
		<p class="mt-3 text-[12px] text-text-muted">
			Want a first-party preset for the app you use? File it as a feature request and we will
			scope it. The bar is: at least three users want it, the protocol is well-documented, and
			adding it does not require us to ship per-vendor SDK code in the build pipeline.
		</p>
	</Card>
</div>

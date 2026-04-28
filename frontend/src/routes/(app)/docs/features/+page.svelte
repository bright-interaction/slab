<script lang="ts">
	import Card from '$lib/components/ui/Card.svelte';
	import { ArrowLeft, ArrowRight } from 'lucide-svelte';
	import { currentSite } from '$lib/stores/currentSite.svelte';

	const siteID = $derived(currentSite.value?.id ?? null);
	const siteHref = $derived((path: string) => (siteID ? `/sites/${siteID}${path}` : '/sites'));
</script>

{#snippet feature(
	title: string,
	desc: string,
	bullets: string[],
	href: string,
	cta: string
)}
	<Card padding="md">
		<h2 class="text-[14px] font-medium text-text-primary">{title}</h2>
		<p class="mt-2 text-[13px] text-text-secondary">{desc}</p>
		<ul class="mt-3 list-disc space-y-1 pl-5 text-[12px] text-text-secondary">
			{#each bullets as b}
				<li>{b}</li>
			{/each}
		</ul>
		<a
			{href}
			class="mt-4 inline-flex items-center gap-1.5 text-[12px] text-text-primary transition-colors hover:text-text-secondary"
		>
			{cta}
			<ArrowRight size={13} strokeWidth={1.75} />
		</a>
	</Card>
{/snippet}

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
			Features
		</h1>
		<p class="text-[13px] text-text-secondary">
			Every admin feature, what it does, where to find it. Links resolve against the currently
			selected site.
		</p>
	</header>

	<div class="mt-8 flex flex-col gap-4">
		{@render feature(
			'Sites and onboarding wizard',
			'Adaptive 5-step wizard. The site type you pick (B2B services, B2C local, personal, blog, agency, e-commerce) drives the rest of the flow.',
			[
				'Auto-generates legal pages: privacy, terms, cookie policy, imprint',
				'Seeds JSON-LD schema, security.txt, robots.txt, llms.txt, sitemap, navigation',
				'Picks a starter kit and applies its components, knowledgebase, and brand defaults',
				'Builder writes a complete Astro project on first build'
			],
			'/sites/new',
			'Create a new site'
		)}

		{@render feature(
			'Pages and blocks',
			'Page hierarchy with a block-based editor. Status is draft or published; PATCH the status to publish.',
			[
				'Page meta enforced: title 30-60, description 120-160',
				'Blocks validated: image blocks need alt + dimensions, CTA blocks need a label',
				'Generic anchor text rejected ("click here", "read more")',
				'Mixed-content http:// URLs rejected on https sites',
				'Move-page action lifts a page across the silo tree'
			],
			siteHref('/pages'),
			'Open Pages'
		)}

		{@render feature(
			'Branding',
			'9 colour slots, curated and self-hosted fonts, GitHub design references, WCAG contrast matrix, scaled mobile or desktop preview.',
			[
				'Colours: Brand (primary, secondary, accent), Surface (background, surface, border), Text (text, muted, on-primary)',
				'Fonts: 12 curated Google Fonts plus woff2 self-hosted upload (2 MB cap, magic-header validated)',
				'Design references: paste a GitHub URL; system fetches package.json, README, tailwind config, globals CSS, 5 representative components',
				'Contrast matrix flags WCAG AA failures with a one-click suggest-fix',
				'Mobile or desktop preview, true scale, live updates'
			],
			siteHref('/branding'),
			'Open Branding'
		)}

		{@render feature(
			'Media library',
			'Image and asset management with folders, alt text, drag-drop upload, base64 and from-URL ingestion.',
			[
				'Folders are lowercase + hyphens, max 32 chars, validator: ^[a-z0-9][a-z0-9-]{0,31}$',
				'Reserved system folder "brand" for brand assets',
				'Unfiled view shows items not in any folder',
				'Drag-drop upload defaults to the active folder',
				'Alt text required by guardrail; SVG rejected for safety; SSRF guards on from-URL'
			],
			siteHref('/media'),
			'Open Media'
		)}

		{@render feature(
			'Components and CSS classes',
			'Reusable building blocks with typed props, plus a global CSS class registry. Both surface to agents through /api/agent/context.',
			[
				'Components: typed props, validated at write time',
				'CSS classes: global registry, agent can read and add classes',
				'Both render inside Astro pages at build time'
			],
			siteHref('/components'),
			'Open Components'
		)}

		{@render feature(
			'Knowledgebase',
			'Reference entries the AI agent reads on every context fetch. Seeded from the brightinteraction.com 140/140 reference site.',
			[
				'24 default entries cover security, SEO, accessibility, brand voice',
				'Add your own entries to encode domain rules',
				'Drives agent decisions: tone, terminology, do-not-say lists'
			],
			siteHref('/knowledgebase'),
			'Open Knowledgebase'
		)}

		{@render feature(
			'Guardrails',
			'Built-in security rules plus user-defined rules. Every agent write passes through this engine.',
			[
				'Built-in: page meta length, image alt, anchor text, mixed content, URL depth',
				'forbid_pattern: regex against block content',
				'allow_block_type: whitelist of block kinds the agent can create',
				'url_depth: max 3 levels (built-in)',
				'Violations return a 422 with a human explanation the agent can act on'
			],
			siteHref('/guardrails'),
			'Open Guardrails'
		)}

		{@render feature(
			'Build',
			'Trigger button, score-history chart, build log list with per-row category grade badges.',
			[
				'Trigger: POST /api/agent/build (admin or agent)',
				'Latest scores donut row: Security, SEO, Performance, Accessibility, Privacy',
				'Per-row badges: Sec / Seo / Per / Acc / Pri',
				'Per-build detail at /build/{buildID}: 5-up donut row, per-category needs-attention and passing sections'
			],
			siteHref('/build'),
			'Open Build'
		)}

		{@render feature(
			'Evaluations',
			'130+ checks across six categories, run after every build. Now merged into the Build page; the per-build detail still lives at /build/{buildID}.',
			[
				'Security: 15 checks (headers, CSP, SRI, mixed content)',
				'SEO: 23 checks (on-page, technical, social, rich results)',
				'GEO: 7 checks (Robots max-snippet, BreadcrumbList, Organization Schema, Content Freshness, AI-friendly formatting, etc.)',
				'Performance: 8 checks (Core Web Vitals, page weight, images)',
				'Accessibility: 14 checks (landmarks, contrast, forms, focus)',
				'Privacy: 6 checks (consent, trackers, pre-consent detection)',
				'13-grade scale: A+ A A- B+ B B- C+ C C- D+ D D- F'
			],
			siteHref('/build'),
			'Open Build'
		)}

		{@render feature(
			'Analytics',
			'Umami-grade dashboard plus a client-side engagement beacon. Server-side nginx tracking is immune to ad blockers.',
			[
				'4-up stat row: visits, unique, identified, live (last 5 min)',
				'Pageviews-over-time chart (hourly for 1d, daily for longer)',
				'Six breakdowns: browsers, OS, devices, countries, languages, UTM',
				'Engagement beacon: time on page, scroll depth, dark-mode share, viewport buckets',
				'Transparency panel: every tracked field with source, every not-tracked field with reason'
			],
			siteHref('/analytics'),
			'Open Analytics'
		)}

		{@render feature(
			'Settings',
			'9 sub-pages covering everything that is not site content.',
			[
				'General: site name, domain, language, defaults',
				'Security: HSTS, CSP, frame options, referrer policy',
				'SEO: meta templates, sitemap, robots, hreflang',
				'Analytics: CookieProof, GA4, Umami, BrightCRM webhook',
				'Allowed Scripts: CSP whitelist for third-party domains',
				'Profile: business info, legal contacts, address (drives Organization JSON-LD)',
				'Nginx: preview the generated server config',
				'Deployment: targets for publishing built sites',
				'Danger Zone: delete this site permanently'
			],
			siteHref('/settings'),
			'Open Settings'
		)}

		{@render feature(
			'Agent keys',
			'Generate, view capabilities, revoke. Every external agent uses one of these.',
			[
				'Format: ask_<64-char hex>',
				'Stored as SHA-256 hash; raw key visible once at creation',
				'Capabilities array: read, write (currently checked)',
				'Scoped to one site by ID; cannot manipulate any other site',
				'One key per use case: dev, prod, CI, your laptop'
			],
			siteHref('/agent-keys'),
			'Open Agent keys'
		)}

		{@render feature(
			'Members and account',
			'Workspace-level user management plus your own account.',
			[
				'/members: invite teammates (admin only), revoke invites, change roles',
				'/signup/{token}: invite acceptance flow',
				'/account: profile, password, sessions (sign out everywhere), preferences (theme)'
			],
			'/account',
			'Open Account'
		)}

		{@render feature(
			'Personalization (CRM round-trip)',
			'Two-way data flow with your CRM. The CRM sees visitor activity (already shipped), looks the visitor up, and pushes knowledge BACK so the built site can greet them, reference what they last read, or gate a CTA on lead score.',
			[
				'POST /t/inbound: HMAC-signed inbound webhook your CRM POSTs to',
				'GET /t/visitor: hydration script reads metadata; cross-origin friendly (server re-derives fingerprint)',
				'Conditional blocks: any block can carry data.condition (e.g. lead_score >= 60); the renderer wraps it in [data-asp-when] and the hydration script reveals it on match',
				'DSL: key op value where op is ==, !=, >, >=, <, <=, in, present',
				'Anonymous tier: last_topic, last_page, lifecycle, lead_score work without identity confirmation',
				'Identified tier: name, email, company gated on identity_confirmed_at within analytics.identity_max_age_days (default 30 days)',
				'Hashed knowledge: keys ending in _hash are matchable but never displayable',
				'Off by default: flip analytics.personalization_enabled to 1 to wire the hydration script'
			],
			siteHref('/settings/analytics'),
			'Open Analytics settings'
		)}
	</div>
</div>

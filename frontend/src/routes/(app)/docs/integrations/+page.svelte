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
		const site = (path: string) => (siteID ? `/sites/${siteID}/${path}` : null);
		return [
			{
				title: 'Cloudflare DNS',
				status: 'shipped',
				body:
					'Wildcard record *.slab.example.com -> 203.0.113.10 covers any built-site slug. Wildcard TLS cert covers the same. No per-site DNS or cert work needed for slugs on the default domain. Custom domains route via Dockyard or your own DNS provider.',
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
					'130+ checks ported from the Chrome extension. Run after every build across Security, SEO, GEO, Performance, Accessibility, Privacy. 13-grade scale, A+ through F. Results expose to the agent through GET /api/agent/evaluation/{buildID} so it can self-correct.',
				linkLabel: 'Build',
				linkHref: site('build')
			},
			{
				title: 'Caddy wildcard host',
				status: 'shipped',
				body:
					'Built sites land at /srv/atomicsite/<slug>/dist via a Caddy bind mount. The wildcard route http://*.slab.example.com serves any slug straight from disk; Local-kind deploy targets land here directly.',
				linkLabel: 'Architecture page',
				linkHref: '/docs/architecture'
			},
			{
				title: 'Forgejo CI deploy pipeline',
				status: 'shipped',
				body:
					'Push to main triggers .forgejo/workflows/deploy-atomicsite.yml: Bun build, Docker build with frontend embedded via go:embed, container push, container restart. End-to-end ~2-3 minutes from push to live.',
				linkLabel: null as never,
				linkHref: null
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
			Platform-level mechanics: DNS, font hosting, the eval engine, the deploy pipeline. For
			user-facing app connectors (CRMs, analytics, banners, automation), see
			<a href="/docs/apps" class="text-text-primary hover:underline">Apps</a>.
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

<script lang="ts">
	import Card from '$lib/components/ui/Card.svelte';
	import {
		Settings as SettingsIcon,
		ShieldCheck,
		Search,
		BarChart3,
		Code2,
		Building2,
		Server,
		Cloud,
		AlertTriangle
	} from 'lucide-svelte';
	import type { Site } from '$lib/api/types';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);
</script>

{#snippet sectionCard(
	href: string,
	title: string,
	subtitle: string,
	IconCmp: typeof SettingsIcon,
	danger: boolean = false
)}
	<a {href} class="block">
		<Card padding="md" hoverable class={danger ? 'border-danger/30' : ''}>
			<div class="flex items-start gap-3">
				<span
					class="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border border-border-light {danger
						? 'text-danger bg-danger/5'
						: 'text-text-secondary bg-bg-elevated'}"
				>
					<IconCmp size={16} strokeWidth={1.5} />
				</span>
				<div class="flex flex-col gap-0.5 min-w-0">
					<p class="text-[13px] font-medium {danger ? 'text-danger' : 'text-text-primary'}">
						{title}
					</p>
					<p class="text-[12px] text-text-muted">{subtitle}</p>
				</div>
			</div>
		</Card>
	</a>
{/snippet}

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex flex-col gap-1.5">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
			Settings
		</h1>
		<p class="text-[13px] text-text-secondary">
			Configure how this site behaves, ships, and surfaces in search.
		</p>
	</header>

	<section class="mt-8 grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
		{@render sectionCard(
			`/sites/${siteID}/settings/general`,
			'General',
			'Site name, domain, language, defaults.',
			SettingsIcon
		)}
		{@render sectionCard(
			`/sites/${siteID}/settings/security`,
			'Security',
			'HSTS, CSP, frame options, referrer policy.',
			ShieldCheck
		)}
		{@render sectionCard(
			`/sites/${siteID}/settings/seo`,
			'SEO',
			'Meta templates, sitemap, robots, hreflang.',
			Search
		)}
		{@render sectionCard(
			`/sites/${siteID}/settings/analytics`,
			'Analytics',
			'CookieProof, GA4, Umami, CRM webhook.',
			BarChart3
		)}
		{@render sectionCard(
			`/sites/${siteID}/settings/allowed-scripts`,
			'Allowed Scripts',
			'CSP whitelist for third-party domains.',
			Code2
		)}
		{@render sectionCard(
			`/sites/${siteID}/settings/profile`,
			'Profile',
			'Business info, legal contacts, address.',
			Building2
		)}
		{@render sectionCard(
			`/sites/${siteID}/settings/nginx`,
			'Nginx',
			'Preview the generated server config.',
			Server
		)}
		{@render sectionCard(
			`/sites/${siteID}/settings/deployment`,
			'Deployment',
			'Targets for publishing built sites.',
			Cloud
		)}
		{@render sectionCard(
			`/sites/${siteID}/settings/danger`,
			'Danger Zone',
			'Delete this site permanently.',
			AlertTriangle,
			true
		)}
	</section>
</div>

<script lang="ts">
	import { onMount } from 'svelte';
	import * as localesApi from '$lib/api/locales';
	import Badge from '$lib/components/ui/Badge.svelte';
	import { Languages } from 'lucide-svelte';

	// Sprint 3 (multilingual v1, 2026-05-22): a locale strip rendered
	// on every page editor. Loads the site's configured locales and
	// the per-page locale overlays so each tab can show its translation
	// status. Clicking a non-default tab navigates to the focused
	// /locales/<locale> editor.
	//
	// Renders nothing when the site has <=1 configured locale; sites
	// that haven't opted into multilingual see no UI noise.

	let { siteID, pageID }: { siteID: string; pageID: string } = $props();

	let siteLocales = $state<localesApi.SiteLocale[]>([]);
	let pageLocales = $state<localesApi.PageLocale[]>([]);
	let loading = $state(true);

	async function load() {
		loading = true;
		try {
			const [sl, pl] = await Promise.all([
				localesApi.listSiteLocales(siteID),
				localesApi.listPageLocales(siteID, pageID)
			]);
			siteLocales = sl;
			pageLocales = pl;
		} catch {
			// Non-fatal: the strip just hides on error.
			siteLocales = [];
			pageLocales = [];
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		void load();
	});

	function statusFor(locale: string): 'default' | 'published' | 'draft' | 'missing' {
		const sl = siteLocales.find((s) => s.locale === locale);
		if (sl?.is_default) return 'default';
		const pl = pageLocales.find((p) => p.locale === locale);
		if (!pl) return 'missing';
		return pl.status === 'published' ? 'published' : 'draft';
	}

	function labelFor(status: ReturnType<typeof statusFor>): string {
		switch (status) {
			case 'default':
				return 'Default';
			case 'published':
				return 'Published';
			case 'draft':
				return 'Draft';
			default:
				return 'Not started';
		}
	}

	function variantFor(status: ReturnType<typeof statusFor>): 'success' | 'warning' | 'info' {
		switch (status) {
			case 'default':
			case 'published':
				return 'success';
			case 'draft':
				return 'warning';
			default:
				return 'info';
		}
	}
</script>

{#if !loading && siteLocales.length > 1}
	<section
		class="mb-4 flex flex-wrap items-center gap-2 rounded-xl border border-border-light bg-bg-surface px-4 py-3"
	>
		<div class="flex items-center gap-2 text-text-secondary">
			<Languages class="h-4 w-4" strokeWidth={1.5} />
			<span class="text-[11px] font-mono uppercase tracking-[0.18em] text-text-muted">Locales</span>
		</div>
		{#each siteLocales as sl (sl.locale)}
			{@const st = statusFor(sl.locale)}
			{#if st === 'default'}
				<span
					class="inline-flex items-center gap-1.5 rounded-lg border border-border-light bg-bg-elevated px-2.5 py-1 text-[12px] font-medium text-text-primary"
				>
					<span class="font-mono">{sl.locale}</span>
					<Badge variant={variantFor(st)}>{labelFor(st)}</Badge>
				</span>
			{:else}
				<a
					href={`/sites/${siteID}/pages/${pageID}/locales/${sl.locale}`}
					class="inline-flex items-center gap-1.5 rounded-lg border border-border-light bg-bg-surface px-2.5 py-1 text-[12px] font-medium text-text-secondary transition-colors hover:bg-bg-elevated hover:text-text-primary"
				>
					<span class="font-mono">{sl.locale}</span>
					<Badge variant={variantFor(st)}>{labelFor(st)}</Badge>
				</a>
			{/if}
		{/each}
	</section>
{/if}

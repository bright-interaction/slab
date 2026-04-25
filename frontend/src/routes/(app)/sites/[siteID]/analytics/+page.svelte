<script lang="ts">
	import { goto } from '$app/navigation';
	import * as analyticsApi from '$lib/api/analytics';
	import type {
		AnalyticsOverview,
		ConversionPath,
		SinceRange,
		VisitSession
	} from '$lib/api/analytics';
	import Card from '$lib/components/ui/Card.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import Select from '$lib/components/ui/Select.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import type { Site } from '$lib/api/types';
	import { CheckCircle2, ChevronRight, BarChart3 } from 'lucide-svelte';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);

	let sinceValue = $state<string>('7d');
	const since = $derived<SinceRange>(
		sinceValue === '30d' || sinceValue === '90d' || sinceValue === 'all'
			? (sinceValue as SinceRange)
			: '7d'
	);

	let overview = $state<AnalyticsOverview | null>(null);
	let overviewLoading = $state(true);
	let overviewError = $state<string | null>(null);

	let identifiedSessions = $state<VisitSession[]>([]);
	let sessionsLoading = $state(true);
	let sessionsError = $state<string | null>(null);

	let conversionPaths = $state<ConversionPath[]>([]);
	let pathsLoading = $state(true);
	let pathsError = $state<string | null>(null);

	let nowTick = $state(Date.now());
	let nowInterval: ReturnType<typeof setInterval> | null = null;

	const sinceOptions = [
		{ value: '7d', label: 'Last 7 days' },
		{ value: '30d', label: 'Last 30 days' },
		{ value: '90d', label: 'Last 90 days' },
		{ value: 'all', label: 'All time' }
	];

	async function loadOverview(id: string, range: SinceRange) {
		overviewLoading = true;
		overviewError = null;
		try {
			overview = await analyticsApi.getOverview(id, range);
		} catch (err) {
			overviewError = err instanceof Error ? err.message : 'Failed to load analytics overview';
			overview = null;
		} finally {
			overviewLoading = false;
		}
	}

	async function loadSessions(id: string) {
		sessionsLoading = true;
		sessionsError = null;
		try {
			const resp = await analyticsApi.listSessions(id, { identified: true, limit: 25 });
			identifiedSessions = resp.sessions ?? [];
		} catch (err) {
			sessionsError = err instanceof Error ? err.message : 'Failed to load sessions';
			identifiedSessions = [];
		} finally {
			sessionsLoading = false;
		}
	}

	async function loadPaths(id: string) {
		pathsLoading = true;
		pathsError = null;
		try {
			const resp = await analyticsApi.listConversionPaths(id, 5);
			conversionPaths = resp.paths ?? [];
		} catch (err) {
			pathsError = err instanceof Error ? err.message : 'Failed to load conversion paths';
			conversionPaths = [];
		} finally {
			pathsLoading = false;
		}
	}

	$effect(() => {
		void loadOverview(siteID, since);
	});

	$effect(() => {
		void loadSessions(siteID);
		void loadPaths(siteID);
	});

	$effect(() => {
		nowInterval = setInterval(() => {
			nowTick = Date.now();
		}, 30_000);
		return () => {
			if (nowInterval) clearInterval(nowInterval);
		};
	});

	function formatRelative(ts: string): string {
		if (!ts) return '';
		const d = new Date(ts);
		if (Number.isNaN(d.getTime())) return '';
		const diffMs = nowTick - d.getTime();
		const sec = Math.max(0, Math.floor(diffMs / 1000));
		if (sec < 60) return `${sec}s ago`;
		const min = Math.floor(sec / 60);
		if (min < 60) return `${min}m ago`;
		const hr = Math.floor(min / 60);
		if (hr < 24) return `${hr}h ago`;
		const days = Math.floor(hr / 24);
		if (days < 30) return `${days}d ago`;
		const months = Math.floor(days / 30);
		if (months < 12) return `${months}mo ago`;
		return `${Math.floor(months / 12)}y ago`;
	}

	function formatNumber(n: number): string {
		if (!Number.isFinite(n)) return '0';
		return new Intl.NumberFormat(undefined).format(n);
	}

	function formatReferer(r: string): string {
		if (!r) return 'Direct';
		try {
			const u = new URL(r);
			return u.hostname.replace(/^www\./, '');
		} catch {
			return r;
		}
	}

	const topPagesMax = $derived(
		overview && overview.top_pages.length > 0
			? Math.max(...overview.top_pages.map((p) => p.count), 1)
			: 1
	);

	const topReferersMax = $derived(
		overview && overview.top_referers.length > 0
			? Math.max(...overview.top_referers.map((r) => r.count), 1)
			: 1
	);

	const isEmpty = $derived(
		!overviewLoading &&
			!overviewError &&
			overview !== null &&
			overview.visits === 0 &&
			overview.unique_visitors === 0
	);

	function goToSettings() {
		goto(`/sites/${siteID}/settings/analytics`);
	}

	function onSessionClick(session: VisitSession) {
		// Detail view deferred. Navigate to a placeholder route the team can
		// replace once the session detail page lands.
		goto(`/sites/${siteID}/analytics/sessions/${session.id}`);
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="flex flex-wrap items-end justify-between gap-4">
		<div>
			<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
				Analytics
			</h1>
			<p class="mt-1 text-[13px] text-text-secondary">
				Visits, identified visitors and conversion paths for this site.
			</p>
		</div>
		<div class="flex items-end gap-2">
			<div class="w-44">
				<Select options={sinceOptions} bind:value={sinceValue} />
			</div>
		</div>
	</header>

	{#if overviewError}
		<Card padding="md" class="mt-6">
			<p class="text-[13px] text-danger">{overviewError}</p>
		</Card>
	{:else if isEmpty}
		<Card padding="none" class="mt-6">
			<EmptyState
				title="No visits yet"
				description="Enable analytics in Settings to start collecting visits. Server-side tracking starts on the next build."
			>
				{#snippet icon()}
					<BarChart3 size={22} strokeWidth={1.5} />
				{/snippet}
				{#snippet action()}
					<Button variant="secondary" onclick={goToSettings}>Open analytics settings</Button>
				{/snippet}
			</EmptyState>
		</Card>
	{:else}
		<section class="mt-6 grid grid-cols-1 gap-3 sm:grid-cols-3">
			<div class="animate-stagger-1">
				<Card padding="md">
					<p class="text-xs text-text-muted">Total visits</p>
					{#if overviewLoading || !overview}
						<Skeleton width="5rem" height="2rem" class="mt-2" />
					{:else}
						<p
							class="mt-1 font-display text-3xl font-extralight tracking-tight text-text-primary"
						>
							{formatNumber(overview.visits)}
						</p>
					{/if}
					<p class="mt-1 text-[11px] text-text-muted">page views in range</p>
				</Card>
			</div>

			<div class="animate-stagger-2">
				<Card padding="md">
					<p class="text-xs text-text-muted">Unique visitors</p>
					{#if overviewLoading || !overview}
						<Skeleton width="5rem" height="2rem" class="mt-2" />
					{:else}
						<p
							class="mt-1 font-display text-3xl font-extralight tracking-tight text-text-primary"
						>
							{formatNumber(overview.unique_visitors)}
						</p>
					{/if}
					<p class="mt-1 text-[11px] text-text-muted">distinct fingerprints</p>
				</Card>
			</div>

			<div class="animate-stagger-3">
				<Card padding="md">
					<p class="text-xs text-text-muted">Identified visitors</p>
					{#if overviewLoading || !overview}
						<Skeleton width="5rem" height="2rem" class="mt-2" />
					{:else}
						<div class="mt-1 flex items-baseline gap-2">
							<p class="font-display text-3xl font-extralight tracking-tight text-text-primary">
								{formatNumber(overview.identified_count)}
							</p>
							{#if overview.unique_visitors > 0}
								<span class="text-[11px] text-text-muted">
									{Math.round((overview.identified_count / overview.unique_visitors) * 100)}%
								</span>
							{/if}
						</div>
					{/if}
					<p class="mt-1 text-[11px] text-text-muted">consented and matched</p>
				</Card>
			</div>
		</section>

		<section class="mt-8 grid grid-cols-1 gap-4 lg:grid-cols-2">
			<Card padding="md">
				<div class="flex items-baseline justify-between">
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Top pages
					</h2>
					{#if overview && overview.top_pages.length > 0}
						<span class="text-[11px] text-text-muted">visits</span>
					{/if}
				</div>
				<div class="mt-3">
					{#if overviewLoading}
						<div class="space-y-2">
							{#each Array(5) as _, i (i)}
								<div class="flex items-center gap-3">
									<Skeleton width="60%" height="0.9rem" />
									<Skeleton width="2rem" height="0.9rem" />
								</div>
							{/each}
						</div>
					{:else if !overview || overview.top_pages.length === 0}
						<p class="py-6 text-center text-[12px] text-text-muted">No page views yet.</p>
					{:else}
						<ul class="space-y-1.5">
							{#each overview.top_pages.slice(0, 10) as p (p.path)}
								{@const pct = Math.max(2, Math.round((p.count / topPagesMax) * 100))}
								<li class="group relative flex items-center gap-3 rounded-md px-2 py-1.5">
									<span
										aria-hidden="true"
										class="absolute inset-y-0 left-0 rounded-md bg-accent/10 transition-all"
										style="width: {pct}%;"
									></span>
									<span
										class="relative z-10 flex-1 truncate font-mono text-[12px] text-text-primary"
									>
										{p.path || '/'}
									</span>
									<span class="relative z-10 text-[12px] tabular-nums text-text-secondary">
										{formatNumber(p.count)}
									</span>
								</li>
							{/each}
						</ul>
					{/if}
				</div>
			</Card>

			<Card padding="md">
				<div class="flex items-baseline justify-between">
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Top referers
					</h2>
					{#if overview && overview.top_referers.length > 0}
						<span class="text-[11px] text-text-muted">visits</span>
					{/if}
				</div>
				<div class="mt-3">
					{#if overviewLoading}
						<div class="space-y-2">
							{#each Array(5) as _, i (i)}
								<div class="flex items-center gap-3">
									<Skeleton width="60%" height="0.9rem" />
									<Skeleton width="2rem" height="0.9rem" />
								</div>
							{/each}
						</div>
					{:else if !overview || overview.top_referers.length === 0}
						<p class="py-6 text-center text-[12px] text-text-muted">
							No referer data yet.
						</p>
					{:else}
						<ul class="space-y-1.5">
							{#each overview.top_referers.slice(0, 10) as r (r.referer || 'direct')}
								{@const pct = Math.max(2, Math.round((r.count / topReferersMax) * 100))}
								<li class="group relative flex items-center gap-3 rounded-md px-2 py-1.5">
									<span
										aria-hidden="true"
										class="absolute inset-y-0 left-0 rounded-md bg-accent/10 transition-all"
										style="width: {pct}%;"
									></span>
									<span class="relative z-10 flex-1 truncate text-[12px] text-text-primary">
										{formatReferer(r.referer)}
									</span>
									<span class="relative z-10 text-[12px] tabular-nums text-text-secondary">
										{formatNumber(r.count)}
									</span>
								</li>
							{/each}
						</ul>
					{/if}
				</div>
			</Card>
		</section>

		<section class="mt-8">
			<Card padding="md">
				<div class="flex items-baseline justify-between">
					<div>
						<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
							Identified sessions
						</h2>
						<p class="mt-1 text-[12px] text-text-muted">
							Visitors who accepted marketing consent and matched to a CRM contact.
						</p>
					</div>
					{#if identifiedSessions.length > 0}
						<Badge variant="default">{identifiedSessions.length}</Badge>
					{/if}
				</div>
				<div class="mt-4">
					{#if sessionsLoading}
						<div class="space-y-2">
							{#each Array(4) as _, i (i)}
								<div class="flex items-center gap-3 rounded-md p-2">
									<Skeleton width="2rem" height="2rem" rounded="full" />
									<div class="flex-1 space-y-1.5">
										<Skeleton width="40%" height="0.9rem" />
										<Skeleton width="25%" height="0.7rem" />
									</div>
								</div>
							{/each}
						</div>
					{:else if sessionsError}
						<p class="py-4 text-[12px] text-danger">{sessionsError}</p>
					{:else if identifiedSessions.length === 0}
						<p class="py-6 text-center text-[12px] text-text-muted">
							No identified sessions yet. They appear once a visitor accepts marketing consent
							and matches a CRM contact.
						</p>
					{:else}
						<ul class="divide-y divide-border-light">
							{#each identifiedSessions as session (session.id)}
								<li>
									<button
										type="button"
										onclick={() => onSessionClick(session)}
										class="flex w-full items-center gap-4 px-2 py-3 text-left transition-colors hover:bg-bg-hover focus-visible:outline-none focus-visible:bg-bg-hover"
									>
										<span
											class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-accent/10 text-[11px] font-medium text-accent"
											aria-hidden="true"
										>
											{(session.email || '?').slice(0, 1).toUpperCase()}
										</span>
										<div class="min-w-0 flex-1">
											<p class="truncate text-[13px] text-text-primary">
												{session.email || 'anonymous'}
											</p>
											<p class="mt-0.5 text-[11px] text-text-muted">
												{session.consent_method
													? `${session.consent_method} consent`
													: 'no consent recorded'}
												{#if session.last_seen_at}
													<span aria-hidden="true"> · </span>{formatRelative(session.last_seen_at)}
												{/if}
											</p>
										</div>
										<div class="flex items-center gap-3 text-[12px] text-text-secondary">
											<span class="tabular-nums">
												{session.page_count} page{session.page_count === 1 ? '' : 's'}
											</span>
											<ChevronRight
												size={14}
												strokeWidth={1.75}
												class="text-text-muted"
											/>
										</div>
									</button>
								</li>
							{/each}
						</ul>
					{/if}
				</div>
			</Card>
		</section>

		<section class="mt-8">
			<Card padding="md">
				<div class="flex items-baseline justify-between">
					<div>
						<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
							Conversion paths
						</h2>
						<p class="mt-1 text-[12px] text-text-muted">
							Page sequence each identified visitor took before consenting.
						</p>
					</div>
				</div>
				<div class="mt-4">
					{#if pathsLoading}
						<div class="space-y-3">
							{#each Array(3) as _, i (i)}
								<Skeleton width="80%" height="1.25rem" />
							{/each}
						</div>
					{:else if pathsError}
						<p class="py-4 text-[12px] text-danger">{pathsError}</p>
					{:else if conversionPaths.length === 0}
						<p class="py-6 text-center text-[12px] text-text-muted">
							No conversion paths yet.
						</p>
					{:else}
						<ul class="space-y-3">
							{#each conversionPaths.slice(0, 5) as path, i (i)}
								<li class="rounded-lg border border-border-light bg-bg-elevated p-3">
									<div class="flex items-center justify-between gap-3">
										<p class="truncate text-[13px] text-text-primary">
											{path.email || 'anonymous'}
										</p>
										<span class="text-[11px] text-text-muted">
											{formatRelative(path.converted_at)}
										</span>
									</div>
									<div
										class="mt-2 flex flex-wrap items-center gap-1.5 overflow-x-auto"
										aria-label="Page sequence"
									>
										{#each path.steps as step, idx (idx)}
											<span
												class="rounded border border-border-light bg-bg-surface px-2 py-0.5 font-mono text-[11px] text-text-secondary"
											>
												{step.path || '/'}
											</span>
											{#if idx < path.steps.length - 1}
												<ChevronRight
													size={12}
													strokeWidth={1.75}
													class="shrink-0 text-text-muted"
												/>
											{/if}
										{/each}
										<CheckCircle2
											size={14}
											strokeWidth={1.75}
											class="ml-1 shrink-0 text-accent"
										/>
									</div>
								</li>
							{/each}
						</ul>
					{/if}
				</div>
			</Card>
		</section>
	{/if}
</div>

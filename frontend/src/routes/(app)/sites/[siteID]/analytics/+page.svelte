<script lang="ts">
	import { goto } from '$app/navigation';
	import * as analyticsApi from '$lib/api/analytics';
	import type {
		AnalyticsOverview,
		ConversionPath,
		GoalAggregate,
		GoalInput,
		GoalMatchType,
		SinceRange,
		TrackedFieldsResponse,
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
		sinceValue === '1d' ||
			sinceValue === '30d' ||
			sinceValue === '90d' ||
			sinceValue === 'all'
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

	let trackedFields = $state<TrackedFieldsResponse | null>(null);
	let trackedExpanded = $state(false);

	// Phase 31.1 conversion goals.
	let goals = $state<GoalAggregate[]>([]);
	let goalsLoading = $state(true);
	let goalsError = $state<string | null>(null);
	let goalUniqueVisitors = $state(0);
	let goalFormOpen = $state(false);
	let goalSaving = $state(false);
	let goalSaveError = $state<string | null>(null);
	let goalDraftSlug = $state('');
	let goalDraftName = $state('');
	let goalDraftMatchType = $state<GoalMatchType>('url_pattern');
	let goalDraftMatchValue = $state('');
	let goalDraftValueCents = $state(0);

	let nowTick = $state(Date.now());
	let nowInterval: ReturnType<typeof setInterval> | null = null;

	const sinceOptions = [
		{ value: '1d', label: 'Last 24 hours' },
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
		void loadGoals(siteID, since);
	});

	$effect(() => {
		void loadSessions(siteID);
		void loadPaths(siteID);
		void loadTrackedFields(siteID);
	});

	async function loadTrackedFields(id: string) {
		try {
			trackedFields = await analyticsApi.getTrackedFields(id);
		} catch {
			trackedFields = null;
		}
	}

	async function loadGoals(id: string, range: SinceRange) {
		goalsLoading = true;
		goalsError = null;
		try {
			const resp = await analyticsApi.getGoalsAnalytics(id, range);
			goals = resp.goals ?? [];
			goalUniqueVisitors = resp.unique_visitors ?? 0;
		} catch (err) {
			goalsError = err instanceof Error ? err.message : 'Failed to load goals';
			goals = [];
		} finally {
			goalsLoading = false;
		}
	}

	function resetGoalDraft() {
		goalDraftSlug = '';
		goalDraftName = '';
		goalDraftMatchType = 'url_pattern';
		goalDraftMatchValue = '';
		goalDraftValueCents = 0;
		goalSaveError = null;
	}

	async function submitGoal(e: Event) {
		e.preventDefault();
		if (goalSaving) return;
		goalSaving = true;
		goalSaveError = null;
		try {
			await analyticsApi.createGoal(siteID, {
				slug: goalDraftSlug.trim(),
				name: goalDraftName.trim(),
				match_type: goalDraftMatchType,
				match_value: goalDraftMatchValue.trim(),
				value_cents: goalDraftValueCents | 0,
				value_currency: 'EUR',
				active: true
			});
			goalFormOpen = false;
			resetGoalDraft();
			await loadGoals(siteID, since);
		} catch (err) {
			goalSaveError = err instanceof Error ? err.message : 'Failed to create goal';
		} finally {
			goalSaving = false;
		}
	}

	async function removeGoal(id: string) {
		if (!confirm('Delete this goal? Recorded conversions for it will be removed.')) return;
		try {
			await analyticsApi.deleteGoal(siteID, id);
			await loadGoals(siteID, since);
		} catch (err) {
			alert(err instanceof Error ? err.message : 'Failed to delete goal');
		}
	}

	function formatDuration(ms: number): string {
		if (!Number.isFinite(ms) || ms <= 0) return '0s';
		const sec = Math.floor(ms / 1000);
		if (sec < 60) return `${sec}s`;
		const m = Math.floor(sec / 60);
		const s = sec % 60;
		if (m < 60) return `${m}m ${s}s`;
		const h = Math.floor(m / 60);
		return `${h}h ${m % 60}m`;
	}

	function bucketLabel(b: string): string {
		// "2026-04-26" or "2026-04-26T14"
		if (b.length === 10) {
			const d = new Date(b + 'T00:00:00Z');
			return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
		}
		if (b.length === 13) {
			return b.slice(11, 13) + ':00';
		}
		return b;
	}

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
		overview && overview.top_pages && overview.top_pages.length > 0
			? Math.max(...overview.top_pages.map((p) => p.count), 1)
			: 1
	);

	const topReferersMax = $derived(
		overview && overview.top_referers && overview.top_referers.length > 0
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
		<section class="mt-6 grid grid-cols-2 gap-3 sm:grid-cols-4">
			<div class="animate-stagger-1">
				<Card padding="md">
					<p class="text-xs text-text-muted">Live now</p>
					{#if overviewLoading || !overview}
						<Skeleton width="3rem" height="2rem" class="mt-2" />
					{:else}
						<div class="mt-1 flex items-baseline gap-2">
							<p
								class="font-display text-3xl font-extralight tracking-tight text-text-primary"
							>
								{formatNumber(overview.live_visitors)}
							</p>
							<span class="inline-flex h-2 w-2 rounded-full bg-emerald-500" aria-hidden="true"></span>
						</div>
					{/if}
					<p class="mt-1 text-[11px] text-text-muted">visitors in last 5 min</p>
				</Card>
			</div>

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

		{#if overview && overview.time_series && overview.time_series.length > 0}
			<section class="mt-8">
				<Card padding="md">
					<div class="flex items-baseline justify-between">
						<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
							Pageviews over time
						</h2>
						<span class="text-[11px] text-text-muted">
							{overview.bucket_unit === 'hour' ? 'per hour' : 'per day'}
						</span>
					</div>
					{@const ts = overview.time_series}
					{@const tsMax = Math.max(...ts.map((p) => p.count), 1)}
					<div class="mt-4 flex h-32 items-end gap-1">
						{#each ts as p (p.bucket)}
							{@const h = Math.max(2, Math.round((p.count / tsMax) * 100))}
							<div class="group relative flex h-full flex-1 flex-col justify-end" title="{bucketLabel(p.bucket)}: {formatNumber(p.count)} visits, {formatNumber(p.unique_count)} unique">
								<div
									class="w-full rounded-t bg-accent/70 transition-colors group-hover:bg-accent"
									style="height: {h}%"
									aria-hidden="true"
								></div>
							</div>
						{/each}
					</div>
					{@const last = ts[ts.length - 1]}
					<div class="mt-2 flex justify-between text-[10px] text-text-muted">
						<span>{ts[0] ? bucketLabel(ts[0].bucket) : ''}</span>
						<span>{last ? bucketLabel(last.bucket) : ''}</span>
					</div>
				</Card>
			</section>
		{/if}

		<section class="mt-8 grid grid-cols-1 gap-4 lg:grid-cols-2">
			<Card padding="md">
				<div class="flex items-baseline justify-between">
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Top pages
					</h2>
					{#if overview && overview.top_pages && overview.top_pages.length > 0}
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
					{:else if !overview || !overview.top_pages || overview.top_pages.length === 0}
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
					{#if overview && overview.top_referers && overview.top_referers.length > 0}
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
					{:else if !overview || !overview.top_referers || overview.top_referers.length === 0}
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

		{#if overview}
			{@const breakdowns = [
				{ key: 'browsers', label: 'Browsers', items: overview.top_browsers ?? [] },
				{ key: 'os', label: 'Operating systems', items: overview.top_os ?? [] },
				{ key: 'devices', label: 'Devices', items: overview.top_devices ?? [] },
				{ key: 'countries', label: 'Countries', items: overview.top_countries ?? [] },
				{ key: 'languages', label: 'Languages', items: overview.top_languages ?? [] },
				{ key: 'utm_sources', label: 'UTM sources', items: overview.top_utm_sources ?? [] }
			].filter((b) => b.items.length > 0)}
			{#if breakdowns.length > 0}
				<section class="mt-8 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
					{#each breakdowns as b (b.key)}
						{@const max = Math.max(...b.items.map((i) => i.count), 1)}
						<Card padding="md">
							<div class="flex items-baseline justify-between">
								<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
									{b.label}
								</h2>
								<span class="text-[11px] text-text-muted">visits</span>
							</div>
							<ul class="mt-3 space-y-1.5">
								{#each b.items.slice(0, 8) as item (item.name)}
									{@const pct = Math.max(2, Math.round((item.count / max) * 100))}
									<li class="group relative flex items-center gap-3 rounded-md px-2 py-1.5">
										<span
											aria-hidden="true"
											class="absolute inset-y-0 left-0 rounded-md bg-accent/10"
											style="width: {pct}%;"
										></span>
										<span class="relative z-10 flex-1 truncate text-[12px] text-text-primary">
											{item.name}
										</span>
										<span class="relative z-10 text-[12px] tabular-nums text-text-secondary">
											{formatNumber(item.count)}
										</span>
									</li>
								{/each}
							</ul>
						</Card>
					{/each}
				</section>
			{/if}
		{/if}

		{#if overview && overview.engagement && overview.engagement.samples > 0}
			{@const eng = overview.engagement}
			<section class="mt-8 grid grid-cols-2 gap-3 sm:grid-cols-4">
				<Card padding="md">
					<p class="text-xs text-text-muted">Avg time on page</p>
					<p class="mt-1 font-display text-2xl font-extralight tracking-tight text-text-primary">
						{formatDuration(eng.avg_time_on_page_ms)}
					</p>
					<p class="mt-1 text-[11px] text-text-muted">
						from {formatNumber(eng.samples)} sample{eng.samples === 1 ? '' : 's'}
					</p>
				</Card>
				<Card padding="md">
					<p class="text-xs text-text-muted">Avg scroll depth</p>
					<p class="mt-1 font-display text-2xl font-extralight tracking-tight text-text-primary">
						{eng.avg_scroll_pct}%
					</p>
					<p class="mt-1 text-[11px] text-text-muted">deepest reached per visit</p>
				</Card>
				<Card padding="md">
					<p class="text-xs text-text-muted">Dark mode</p>
					<p class="mt-1 font-display text-2xl font-extralight tracking-tight text-text-primary">
						{eng.dark_pct}%
					</p>
					<p class="mt-1 text-[11px] text-text-muted">visitors prefer dark</p>
				</Card>
				<Card padding="md">
					<p class="text-xs text-text-muted">Reduced motion</p>
					<p class="mt-1 font-display text-2xl font-extralight tracking-tight text-text-primary">
						{eng.reduced_motion_pct}%
					</p>
					<p class="mt-1 text-[11px] text-text-muted">accessibility preference</p>
				</Card>
			</section>

			{#if eng.by_path.length > 0 || eng.viewport_buckets.length > 0}
				<section class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-2">
					{#if eng.by_path.length > 0}
						<Card padding="md">
							<div class="flex items-baseline justify-between">
								<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
									Engagement by page
								</h2>
								<span class="text-[11px] text-text-muted">avg time / scroll</span>
							</div>
							<ul class="mt-3 space-y-1.5">
								{#each eng.by_path.slice(0, 8) as row (row.path)}
									<li class="flex items-center gap-3 rounded-md px-2 py-1.5">
										<span class="flex-1 truncate font-mono text-[12px] text-text-primary">
											{row.path || '/'}
										</span>
										<span class="text-[12px] tabular-nums text-text-secondary">
											{formatDuration(row.avg_time_on_page_ms)}
										</span>
										<span class="w-12 text-right text-[12px] tabular-nums text-text-secondary">
											{row.avg_scroll_pct}%
										</span>
									</li>
								{/each}
							</ul>
						</Card>
					{/if}
					{#if eng.viewport_buckets.length > 0}
						{@const vpMax = Math.max(...eng.viewport_buckets.map((b) => b.count), 1)}
						<Card padding="md">
							<div class="flex items-baseline justify-between">
								<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
									Top viewport widths
								</h2>
								<span class="text-[11px] text-text-muted">visits</span>
							</div>
							<ul class="mt-3 space-y-1.5">
								{#each eng.viewport_buckets.slice(0, 8) as v (v.name)}
									{@const pct = Math.max(2, Math.round((v.count / vpMax) * 100))}
									<li class="group relative flex items-center gap-3 rounded-md px-2 py-1.5">
										<span
											aria-hidden="true"
											class="absolute inset-y-0 left-0 rounded-md bg-accent/10"
											style="width: {pct}%;"
										></span>
										<span class="relative z-10 flex-1 truncate text-[12px] text-text-primary">
											{v.name}
										</span>
										<span class="relative z-10 text-[12px] tabular-nums text-text-secondary">
											{formatNumber(v.count)}
										</span>
									</li>
								{/each}
							</ul>
						</Card>
					{/if}
				</section>
			{/if}
		{/if}

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
				<div class="flex items-baseline justify-between gap-3">
					<div>
						<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
							Goals
						</h2>
						<p class="mt-1 text-[12px] text-text-muted">
							Conversions tracked over the active range. Match a URL pattern,
							fire <code class="font-mono">window.atomic.track()</code>, or
							hook a form submission.
						</p>
					</div>
					<Button
						size="sm"
						variant="secondary"
						onclick={() => {
							goalFormOpen = !goalFormOpen;
							if (!goalFormOpen) resetGoalDraft();
						}}
					>
						{goalFormOpen ? 'Cancel' : 'New goal'}
					</Button>
				</div>

				{#if goalFormOpen}
					<form
						class="mt-4 grid gap-3 rounded-lg border border-border-light bg-bg-elevated p-4 sm:grid-cols-2"
						onsubmit={submitGoal}
					>
						<label class="flex flex-col gap-1 text-[12px] text-text-muted">
							Slug
							<input
								type="text"
								required
								pattern="[a-z0-9][a-z0-9_-]*"
								maxlength="63"
								placeholder="lead"
								class="rounded border border-border-light bg-bg-surface px-2 py-1 text-[13px] text-text-primary"
								bind:value={goalDraftSlug}
							/>
						</label>
						<label class="flex flex-col gap-1 text-[12px] text-text-muted">
							Name
							<input
								type="text"
								required
								maxlength="120"
								placeholder="Lead captured"
								class="rounded border border-border-light bg-bg-surface px-2 py-1 text-[13px] text-text-primary"
								bind:value={goalDraftName}
							/>
						</label>
						<label class="flex flex-col gap-1 text-[12px] text-text-muted">
							Match type
							<select
								class="rounded border border-border-light bg-bg-surface px-2 py-1 text-[13px] text-text-primary"
								bind:value={goalDraftMatchType}
							>
								<option value="url_pattern">URL pattern</option>
								<option value="event_name">Event name (atomic.track)</option>
								<option value="form_submit">Form submit</option>
							</select>
						</label>
						<label class="flex flex-col gap-1 text-[12px] text-text-muted">
							Match value
							<input
								type="text"
								required
								maxlength="1024"
								placeholder={goalDraftMatchType === 'url_pattern'
									? '/thank-you/*'
									: goalDraftMatchType === 'event_name'
										? 'signup'
										: 'form_id'}
								class="rounded border border-border-light bg-bg-surface px-2 py-1 font-mono text-[13px] text-text-primary"
								bind:value={goalDraftMatchValue}
							/>
						</label>
						<label class="flex flex-col gap-1 text-[12px] text-text-muted sm:col-span-2">
							Value (cents, optional)
							<input
								type="number"
								min="0"
								step="1"
								placeholder="0"
								class="rounded border border-border-light bg-bg-surface px-2 py-1 text-[13px] text-text-primary"
								bind:value={goalDraftValueCents}
							/>
						</label>
						{#if goalSaveError}
							<p class="text-[12px] text-danger sm:col-span-2">{goalSaveError}</p>
						{/if}
						<div class="flex justify-end gap-2 sm:col-span-2">
							<Button
								size="sm"
								variant="ghost"
								onclick={() => {
									goalFormOpen = false;
									resetGoalDraft();
								}}
								type="button"
							>
								Cancel
							</Button>
							<Button size="sm" variant="primary" type="submit" disabled={goalSaving}>
								{goalSaving ? 'Saving...' : 'Create goal'}
							</Button>
						</div>
					</form>
				{/if}

				<div class="mt-4">
					{#if goalsLoading}
						<div class="space-y-2">
							{#each Array(3) as _, i (i)}
								<Skeleton width="100%" height="2.5rem" />
							{/each}
						</div>
					{:else if goalsError}
						<p class="py-4 text-[12px] text-danger">{goalsError}</p>
					{:else if goals.length === 0}
						<EmptyState
							title="No goals yet"
							description="Define a goal to start measuring conversion rate."
							icon={BarChart3}
						/>
					{:else}
						<table class="w-full text-[13px]">
							<thead>
								<tr class="border-b border-border-light text-[11px] font-mono uppercase tracking-[0.15em] text-text-muted">
									<th class="py-2 text-left font-normal">Goal</th>
									<th class="py-2 text-left font-normal">Match</th>
									<th class="py-2 text-right font-normal">Conv</th>
									<th class="py-2 text-right font-normal">Unique</th>
									<th class="py-2 text-right font-normal">Rate</th>
									<th class="py-2 text-right font-normal" aria-label="Actions"></th>
								</tr>
							</thead>
							<tbody>
								{#each goals as g (g.id)}
									<tr class="border-b border-border-light/50">
										<td class="py-2">
											<div class="font-medium text-text-primary">{g.name}</div>
											<div class="font-mono text-[11px] text-text-muted">{g.slug}</div>
										</td>
										<td class="py-2 font-mono text-[11px] text-text-secondary">
											<Badge>{g.match_type}</Badge>
											<span class="ml-1">{g.match_value}</span>
										</td>
										<td class="py-2 text-right tabular-nums">{formatNumber(g.conversions)}</td>
										<td class="py-2 text-right tabular-nums">{formatNumber(g.unique_converters)}</td>
										<td class="py-2 text-right tabular-nums">
											{g.conversion_rate_pct.toFixed(2)}%
										</td>
										<td class="py-2 text-right">
											<button
												type="button"
												class="text-[11px] text-text-muted hover:text-danger"
												onclick={() => removeGoal(g.id)}
												aria-label="Delete {g.name}"
											>
												Delete
											</button>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
						{#if goalUniqueVisitors > 0}
							<p class="mt-3 text-[11px] text-text-muted">
								Rate is conversions / unique visitors over the same range
								({formatNumber(goalUniqueVisitors)} unique).
							</p>
						{/if}
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

	{#if trackedFields}
		<section class="mt-12">
			<Card padding="md">
				<button
					type="button"
					class="flex w-full items-baseline justify-between text-left"
					onclick={() => (trackedExpanded = !trackedExpanded)}
					aria-expanded={trackedExpanded}
				>
					<div>
						<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
							What we track
						</h2>
						<p class="mt-1 text-[12px] text-text-muted">
							Server-side, no cookies, no IP storage.
							{trackedFields.tracked.length} fields recorded,
							{trackedFields.not_tracked.length} explicitly never recorded.
							Click to {trackedExpanded ? 'collapse' : 'expand'}.
						</p>
					</div>
					<ChevronRight
						size={14}
						strokeWidth={1.75}
						class="shrink-0 text-text-muted transition-transform {trackedExpanded
							? 'rotate-90'
							: ''}"
					/>
				</button>

				{#if trackedExpanded}
					<div class="mt-5 grid grid-cols-1 gap-6 lg:grid-cols-2">
						<div>
							<h3 class="text-[12px] font-medium text-text-primary">Tracked</h3>
							<ul class="mt-2 space-y-2">
								{#each trackedFields.tracked as t (t.field)}
									<li class="rounded-md border border-border-light bg-bg-elevated p-2.5">
										<p class="font-mono text-[12px] text-text-primary">{t.field}</p>
										<p class="mt-0.5 text-[11px] text-text-muted">
											<span class="font-medium">Stored:</span>
											{t.stored}
										</p>
										<p class="text-[11px] text-text-muted">
											<span class="font-medium">Source:</span>
											{t.source}
										</p>
										<p class="text-[11px] text-text-muted">
											<span class="font-medium">Why:</span>
											{t.purpose}
										</p>
									</li>
								{/each}
							</ul>
						</div>
						<div>
							<h3 class="text-[12px] font-medium text-text-primary">Never tracked</h3>
							<ul class="mt-2 space-y-2">
								{#each trackedFields.not_tracked as t (t.field)}
									<li class="rounded-md border border-border-light bg-bg-elevated p-2.5">
										<p class="font-mono text-[12px] text-text-primary">{t.field}</p>
										<p class="mt-0.5 text-[11px] text-text-muted">{t.reason}</p>
									</li>
								{/each}
							</ul>
						</div>
					</div>
				{/if}
			</Card>
		</section>
	{/if}
</div>

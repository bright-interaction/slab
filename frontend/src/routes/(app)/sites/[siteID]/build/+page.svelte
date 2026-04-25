<script lang="ts">
	import { goto } from '$app/navigation';
	import * as buildsApi from '$lib/api/builds';
	import * as evaluationsApi from '$lib/api/evaluations';
	import {
		pollBuild,
		getState,
		clearBuild,
		type BuildStatusState,
		type BuildStatusValue
	} from '$lib/stores/buildStatus.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import GradeBadge from '$lib/components/ui/GradeBadge.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import type { Site, Evaluation } from '$lib/api/types';

	let { data }: { data: { site: Site } } = $props();

	const siteID = $derived(data.site.id);

	let triggering = $state(false);
	let activeBuildID = $state<string | null>(null);
	let stopFn = $state<(() => void) | null>(null);
	let nowTick = $state(Date.now());
	let nowInterval: ReturnType<typeof setInterval> | null = null;

	let logEl = $state<HTMLPreElement | null>(null);
	let userScrolledUp = $state(false);
	let lastTerminalReported = $state<BuildStatusValue | null>(null);

	let recentLoading = $state(true);
	let recentRows = $state<RecentBuildRow[]>([]);
	let recentError = $state<string | null>(null);

	type Grade = 'A+' | 'A' | 'B+' | 'B' | 'C' | 'D' | 'F';
	const validGrades: Grade[] = ['A+', 'A', 'B+', 'B', 'C', 'D', 'F'];
	const gradeRank: Record<Grade, number> = {
		'A+': 0,
		A: 1,
		'B+': 2,
		B: 3,
		C: 4,
		D: 5,
		F: 6
	};

	function asGrade(value: string): Grade | null {
		return (validGrades as string[]).includes(value) ? (value as Grade) : null;
	}

	interface RecentBuildRow {
		buildID: string;
		created_at: string;
		composite: Grade | null;
		categoryCount: number;
	}

	function compositeGrade(evals: Evaluation[]): Grade | null {
		let worst: Grade | null = null;
		for (const e of evals) {
			const g = asGrade(e.grade);
			if (!g) continue;
			if (worst === null || gradeRank[g] > gradeRank[worst]) worst = g;
		}
		return worst;
	}

	function groupRecentBuilds(evals: Evaluation[]): RecentBuildRow[] {
		const map = new Map<string, Evaluation[]>();
		for (const e of evals) {
			if (!e.build_id) continue;
			const list = map.get(e.build_id) ?? [];
			list.push(e);
			map.set(e.build_id, list);
		}
		const rows: RecentBuildRow[] = [];
		for (const [buildID, group] of map.entries()) {
			const newest = group.reduce((acc, cur) =>
				new Date(cur.created_at).getTime() > new Date(acc.created_at).getTime() ? cur : acc
			);
			rows.push({
				buildID,
				created_at: newest.created_at,
				composite: compositeGrade(group),
				categoryCount: group.length
			});
		}
		rows.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
		return rows.slice(0, 10);
	}

	async function loadRecent() {
		recentLoading = true;
		recentError = null;
		try {
			const evals = await evaluationsApi.listBySite(siteID);
			recentRows = groupRecentBuilds(evals);
		} catch (err) {
			recentError = err instanceof Error ? err.message : 'Failed to load build history';
			recentRows = [];
		} finally {
			recentLoading = false;
		}
	}

	$effect(() => {
		void loadRecent();
	});

	$effect(() => {
		nowInterval = setInterval(() => {
			nowTick = Date.now();
		}, 1_000);
		return () => {
			if (nowInterval) clearInterval(nowInterval);
		};
	});

	const buildState = $derived.by<BuildStatusState | null>(() => {
		// Touch nowTick so this re-runs every second while polling.
		void nowTick;
		if (!activeBuildID) return null;
		return getState(activeBuildID);
	});

	const elapsedSeconds = $derived.by(() => {
		if (!buildState) return 0;
		return Math.max(0, Math.floor((nowTick - buildState.started_at) / 1000));
	});

	$effect(() => {
		if (!buildState) return;
		if (lastTerminalReported === buildState.status) return;
		if (buildState.status === 'success') {
			lastTerminalReported = 'success';
			toast.success('Build succeeded. View evaluations to see results.');
			void loadRecent();
		} else if (buildState.status === 'failed') {
			lastTerminalReported = 'failed';
			toast.error(buildState.error || 'Build failed.');
			void loadRecent();
		} else if (buildState.status === 'timeout') {
			lastTerminalReported = 'timeout';
			toast.error('Build timed out after 5 minutes.');
		}
	});

	$effect(() => {
		if (!buildState || !logEl) return;
		// Touch build_log to subscribe.
		const _ = buildState.build_log;
		void _;
		if (!userScrolledUp) {
			queueMicrotask(() => {
				if (logEl) logEl.scrollTop = logEl.scrollHeight;
			});
		}
	});

	function onLogScroll() {
		if (!logEl) return;
		const distFromBottom = logEl.scrollHeight - (logEl.scrollTop + logEl.clientHeight);
		userScrolledUp = distFromBottom > 12;
	}

	async function onTrigger() {
		if (triggering) return;
		triggering = true;
		try {
			const resp = await buildsApi.triggerBuild(siteID);
			activeBuildID = resp.build_id;
			lastTerminalReported = null;
			userScrolledUp = false;
			if (stopFn) stopFn();
			stopFn = pollBuild(siteID, resp.build_id);
			toast.info('Build started.');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Failed to trigger build.');
		} finally {
			triggering = false;
		}
	}

	function onClearActive() {
		if (stopFn) stopFn();
		stopFn = null;
		if (activeBuildID) clearBuild(activeBuildID);
		activeBuildID = null;
		userScrolledUp = false;
		lastTerminalReported = null;
	}

	$effect(() => {
		return () => {
			if (stopFn) stopFn();
		};
	});

	function statusVariant(s: BuildStatusValue): 'success' | 'warning' | 'danger' | 'default' {
		if (s === 'success') return 'success';
		if (s === 'building' || s === 'pending') return 'warning';
		if (s === 'failed' || s === 'timeout') return 'danger';
		return 'default';
	}

	function formatDate(ts: string): string {
		if (!ts) return '';
		const d = new Date(ts);
		if (Number.isNaN(d.getTime())) return '';
		return d.toLocaleString(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}

	const isTerminal = $derived.by(() => {
		if (!buildState) return false;
		return (
			buildState.status === 'success' ||
			buildState.status === 'failed' ||
			buildState.status === 'timeout'
		);
	});
</script>

<div class="mx-auto max-w-5xl px-6 py-8">
	<header class="mb-6">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">Build</h1>
		<p class="mt-1 text-[13px] text-text-secondary">
			Compile your site to a static Astro project.
		</p>
	</header>

	<Card padding="md">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<div>
				<h2 class="text-[13px] font-medium text-text-primary">Trigger build</h2>
				<p class="mt-1 text-[12px] text-text-muted">
					Generates the Astro workspace, runs bun build, and scores the output.
				</p>
			</div>
			<Button
				variant="primary"
				loading={triggering || (buildState !== null && !isTerminal)}
				disabled={triggering || (buildState !== null && !isTerminal)}
				onclick={onTrigger}
			>
				{buildState && !isTerminal ? 'Building...' : 'Trigger build'}
			</Button>
		</div>
	</Card>

	{#if buildState}
		<section class="mt-6">
			<Card padding="md">
				<div class="flex flex-wrap items-center justify-between gap-3">
					<div class="flex items-center gap-3">
						<Badge variant={statusVariant(buildState.status)} dot>
							{buildState.status}
						</Badge>
						<span class="text-[12px] text-text-muted">
							{elapsedSeconds}s elapsed
						</span>
						<span class="text-[12px] text-text-muted">
							{buildState.pages_built} page{buildState.pages_built === 1 ? '' : 's'} built
						</span>
					</div>
					<div class="flex items-center gap-2">
						{#if isTerminal && buildState.status === 'success'}
							<Button
								variant="secondary"
								onclick={() => goto(`/sites/${siteID}/evaluations/${buildState!.buildID}`)}
							>
								View evaluation
							</Button>
						{/if}
						{#if isTerminal}
							<Button variant="ghost" onclick={onClearActive}>Dismiss</Button>
						{/if}
					</div>
				</div>

				{#if buildState.error}
					<p class="mt-3 text-[12px] text-danger">{buildState.error}</p>
				{/if}

				<pre
					bind:this={logEl}
					onscroll={onLogScroll}
					class="mt-4 rounded-lg border border-border-light bg-bg-elevated p-3 font-mono text-xs whitespace-pre-wrap max-h-[60vh] overflow-auto text-text-primary"
				>{buildState.build_log || 'Waiting for output...'}</pre>
			</Card>
		</section>
	{/if}

	<section class="mt-8">
		<div class="flex items-baseline justify-between">
			<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
				Build history
			</h2>
			{#if recentRows.length > 0}
				<a
					href={`/sites/${siteID}/evaluations`}
					class="text-[12px] text-text-muted hover:text-text-primary transition-colors"
				>
					View all
				</a>
			{/if}
		</div>

		<div class="mt-3">
			{#if recentLoading}
				<div class="space-y-2">
					{#each Array(3) as _, i (i)}
						<Card padding="sm">
							<div class="flex items-center gap-3">
								<Skeleton width="2.25rem" height="2.25rem" rounded="full" />
								<div class="flex-1 space-y-1.5">
									<Skeleton width="40%" height="0.9rem" />
									<Skeleton width="25%" height="0.7rem" />
								</div>
							</div>
						</Card>
					{/each}
				</div>
			{:else if recentError}
				<Card padding="md">
					<p class="text-[13px] text-danger">{recentError}</p>
				</Card>
			{:else if recentRows.length === 0}
				<Card padding="none">
					<EmptyState
						title="No builds yet"
						description="Trigger your first build to compile and score this site."
					/>
				</Card>
			{:else}
				<div class="space-y-2">
					{#each recentRows as row (row.buildID)}
						<Card padding="sm" hoverable>
							<div class="flex items-center gap-4">
								{#if row.composite}
									<GradeBadge grade={row.composite} size="sm" />
								{:else}
									<span
										class="inline-flex h-9 w-9 items-center justify-center rounded-2xl bg-bg-elevated text-text-muted text-base"
									>
										?
									</span>
								{/if}
								<div class="flex-1 min-w-0">
									<p class="text-[13px] font-medium text-text-primary">
										{formatDate(row.created_at)}
									</p>
									<p class="text-[11px] text-text-muted">
										{row.categoryCount} categor{row.categoryCount === 1 ? 'y' : 'ies'} scored
									</p>
								</div>
								<a
									href={`/sites/${siteID}/evaluations/${row.buildID}`}
									class="text-[12px] text-text-secondary hover:text-text-primary transition-colors"
								>
									View
								</a>
							</div>
						</Card>
					{/each}
				</div>
			{/if}
		</div>
	</section>
</div>

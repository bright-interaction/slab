<script lang="ts">
	import { goto } from '$app/navigation';
	import * as evaluationsApi from '$lib/api/evaluations';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Badge from '$lib/components/ui/Badge.svelte';
	import GradeBadge from '$lib/components/ui/GradeBadge.svelte';
	import EmptyState from '$lib/components/ui/EmptyState.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import CategoryDonut from '$lib/components/evaluations/CategoryDonut.svelte';
	import type { Site, Evaluation } from '$lib/api/types';

	let { data }: { data: { site: Site } } = $props();
	const siteID = $derived(data.site.id);

	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let evaluations = $state<Evaluation[]>([]);

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

	const CATEGORIES = ['security', 'seo', 'performance', 'accessibility', 'privacy'] as const;

	interface BuildRow {
		buildID: string;
		created_at: string;
		evaluations: Evaluation[];
		composite: Grade | null;
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

	const rows = $derived.by<BuildRow[]>(() => {
		const map = new Map<string, Evaluation[]>();
		for (const e of evaluations) {
			if (!e.build_id) continue;
			const list = map.get(e.build_id) ?? [];
			list.push(e);
			map.set(e.build_id, list);
		}
		const out: BuildRow[] = [];
		for (const [buildID, group] of map.entries()) {
			const newest = group.reduce((acc, cur) =>
				new Date(cur.created_at).getTime() > new Date(acc.created_at).getTime() ? cur : acc
			);
			out.push({
				buildID,
				created_at: newest.created_at,
				evaluations: group,
				composite: compositeGrade(group)
			});
		}
		out.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
		return out;
	});

	const latestBuild = $derived<BuildRow | null>(rows[0] ?? null);

	function evalForCategory(row: BuildRow | null, cat: string): Evaluation | null {
		if (!row) return null;
		return row.evaluations.find((e) => e.category === cat) ?? null;
	}

	async function load() {
		loading = true;
		loadError = null;
		try {
			evaluations = await evaluationsApi.listBySite(siteID);
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load evaluations';
			evaluations = [];
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load();
	});

	function gradeBadgeVariant(g: string): 'success' | 'warning' | 'danger' | 'default' {
		const grade = asGrade(g);
		if (!grade) return 'default';
		if (grade === 'A+' || grade === 'A' || grade === 'B+' || grade === 'B') return 'success';
		if (grade === 'C') return 'warning';
		return 'danger';
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

	function categoryGrade(row: BuildRow, cat: string): string {
		const ev = row.evaluations.find((e) => e.category === cat);
		return ev?.grade ?? '-';
	}
</script>

<div class="mx-auto max-w-7xl px-6 py-8">
	<header class="mb-6">
		<h1 class="font-display text-3xl font-extralight tracking-tight text-text-primary">
			Evaluations
		</h1>
		<p class="mt-1 text-[13px] text-text-secondary">
			Security, SEO, performance, accessibility, and privacy scores for every build.
		</p>
	</header>

	<section class="mb-8">
		<Card padding="md">
			<div class="flex items-baseline justify-between">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Latest scores
				</h2>
				{#if latestBuild}
					<span class="text-[11px] text-text-muted">{formatDate(latestBuild.created_at)}</span>
				{/if}
			</div>
			<div class="mt-5">
				{#if loading}
					<div class="grid grid-cols-2 gap-6 sm:grid-cols-3 lg:grid-cols-5">
						{#each CATEGORIES as cat (cat)}
							<div class="flex flex-col items-center gap-2">
								<Skeleton width="8rem" height="8rem" rounded="full" />
								<Skeleton width="3rem" height="0.7rem" />
							</div>
						{/each}
					</div>
				{:else if !latestBuild}
					<p class="py-6 text-center text-[12px] text-text-muted">
						No build scores yet.
					</p>
				{:else}
					<div class="grid grid-cols-2 gap-6 sm:grid-cols-3 lg:grid-cols-5">
						{#each CATEGORIES as cat (cat)}
							<CategoryDonut category={cat} evaluation={evalForCategory(latestBuild, cat)} />
						{/each}
					</div>
				{/if}
			</div>
		</Card>
	</section>

	<section>
		<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Builds</h2>

		<div class="mt-3">
			{#if loading}
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
			{:else if loadError}
				<Card padding="md">
					<p class="text-[13px] text-danger">{loadError}</p>
				</Card>
			{:else if rows.length === 0}
				<Card padding="none">
					<EmptyState
						title="No evaluations yet"
						description="Trigger your first build to score this site."
					>
						{#snippet action()}
							<Button variant="primary" onclick={() => goto(`/sites/${siteID}/build`)}>
								Trigger your first build
							</Button>
						{/snippet}
					</EmptyState>
				</Card>
			{:else}
				<div class="space-y-2">
					{#each rows as row (row.buildID)}
						<Card padding="sm" hoverable>
							<div class="flex flex-wrap items-center gap-4">
								{#if row.composite}
									<GradeBadge grade={row.composite} size="sm" />
								{:else}
									<span
										class="inline-flex h-9 w-9 items-center justify-center rounded-2xl bg-bg-elevated text-text-muted text-base"
									>
										?
									</span>
								{/if}
								<div class="min-w-0 flex-1">
									<p class="text-[13px] font-medium text-text-primary">
										{formatDate(row.created_at)}
									</p>
									<p class="text-[11px] text-text-muted font-mono truncate">
										{row.buildID}
									</p>
								</div>
								<div class="flex flex-wrap gap-1.5">
									{#each CATEGORIES as cat (cat)}
										{@const g = categoryGrade(row, cat)}
										<Badge variant={gradeBadgeVariant(g)}>
											<span class="capitalize">{cat.slice(0, 3)}</span>
											<span class="ml-1 font-mono">{g}</span>
										</Badge>
									{/each}
								</div>
								<a
									href={`/sites/${siteID}/evaluations/${row.buildID}`}
									class="text-[12px] text-text-secondary hover:text-text-primary transition-colors"
								>
									Detail
								</a>
							</div>
						</Card>
					{/each}
				</div>
			{/if}
		</div>
	</section>
</div>

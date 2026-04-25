<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import * as evaluationsApi from '$lib/api/evaluations';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import GradeBadge from '$lib/components/ui/GradeBadge.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import CategoryGradeCard from '$lib/components/evaluations/CategoryGradeCard.svelte';
	import ScoreHistoryChart, {
		type BuildHistoryEntry
	} from '$lib/components/evaluations/ScoreHistoryChart.svelte';
	import type { Site, Evaluation } from '$lib/api/types';

	let { data }: { data: { site: Site } } = $props();
	const siteID = $derived(data.site.id);
	const buildID = $derived($page.params.buildID ?? '');

	const CATEGORIES = ['security', 'seo', 'performance', 'accessibility', 'privacy'] as const;

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

	let loading = $state(true);
	let loadError = $state<string | null>(null);
	let buildEvals = $state<Evaluation[]>([]);
	let allEvals = $state<Evaluation[]>([]);

	async function load(id: string) {
		loading = true;
		loadError = null;
		try {
			const [forBuild, forSite] = await Promise.all([
				evaluationsApi.listByBuild(siteID, id),
				evaluationsApi.listBySite(siteID).catch(() => [] as Evaluation[])
			]);
			buildEvals = forBuild;
			allEvals = forSite;
		} catch (err) {
			loadError = err instanceof Error ? err.message : 'Failed to load evaluation';
			buildEvals = [];
			allEvals = [];
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		void load(buildID);
	});

	const compositeGrade = $derived.by<Grade | null>(() => {
		let worst: Grade | null = null;
		for (const e of buildEvals) {
			const g = asGrade(e.grade);
			if (!g) continue;
			if (worst === null || gradeRank[g] > gradeRank[worst]) worst = g;
		}
		return worst;
	});

	const buildCreatedAt = $derived(buildEvals[0]?.created_at ?? '');

	const evalByCategory = $derived.by<Record<string, Evaluation | null>>(() => {
		const map: Record<string, Evaluation | null> = {};
		for (const cat of CATEGORIES) {
			map[cat] = buildEvals.find((e) => e.category === cat) ?? null;
		}
		return map;
	});

	const chartData = $derived.by<BuildHistoryEntry[]>(() => {
		const map = new Map<string, Evaluation[]>();
		for (const e of allEvals) {
			if (!e.build_id) continue;
			const list = map.get(e.build_id) ?? [];
			list.push(e);
			map.set(e.build_id, list);
		}
		const out: BuildHistoryEntry[] = [];
		for (const [bid, group] of map.entries()) {
			const newest = group.reduce((acc, cur) =>
				new Date(cur.created_at).getTime() > new Date(acc.created_at).getTime() ? cur : acc
			);
			out.push({ build_id: bid, created_at: newest.created_at, evaluations: group });
		}
		out.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime());
		return out.slice(0, 10);
	});

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
</script>

<div class="mx-auto max-w-6xl px-6 py-8">
	<div class="mb-6 flex flex-wrap items-start justify-between gap-4">
		<div>
			<a
				href={`/sites/${siteID}/evaluations`}
				class="text-[12px] text-text-muted hover:text-text-primary transition-colors"
			>
				&larr; All evaluations
			</a>
			<h1 class="mt-2 font-display text-3xl font-extralight tracking-tight text-text-primary">
				Build evaluation
			</h1>
			<p class="mt-1 text-[12px] text-text-muted font-mono">{buildID}</p>
			{#if buildCreatedAt}
				<p class="mt-0.5 text-[12px] text-text-secondary">{formatDate(buildCreatedAt)}</p>
			{/if}
		</div>
		<div class="flex items-center gap-3">
			<div class="text-right">
				<p class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Composite
				</p>
				<div class="mt-1">
					{#if compositeGrade}
						<GradeBadge grade={compositeGrade} size="lg" />
					{:else}
						<span
							class="inline-flex h-20 w-20 items-center justify-center rounded-2xl bg-bg-elevated text-text-muted text-4xl"
						>
							?
						</span>
					{/if}
				</div>
			</div>
			<Button variant="secondary" onclick={() => goto(`/sites/${siteID}/build`)}>
				Trigger new build
			</Button>
		</div>
	</div>

	{#if loading}
		<div class="grid gap-4 lg:grid-cols-2">
			{#each Array(4) as _, i (i)}
				<Skeleton width="100%" height="180px" />
			{/each}
		</div>
	{:else if loadError}
		<Card padding="md">
			<p class="text-[13px] text-danger">{loadError}</p>
		</Card>
	{:else}
		<section class="grid gap-4 lg:grid-cols-2">
			{#each CATEGORIES as cat (cat)}
				<CategoryGradeCard
					category={cat}
					evaluation={evalByCategory[cat] ?? null}
					{siteID}
					defaultExpanded={false}
				/>
			{/each}
		</section>

		<section class="mt-8">
			<Card padding="md">
				<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
					Score history (context)
				</h2>
				<div class="mt-3">
					<ScoreHistoryChart builds={chartData} height={180} />
				</div>
			</Card>
		</section>
	{/if}
</div>

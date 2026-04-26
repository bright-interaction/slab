<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import * as evaluationsApi from '$lib/api/evaluations';
	import Card from '$lib/components/ui/Card.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import GradeBadge from '$lib/components/ui/GradeBadge.svelte';
	import Skeleton from '$lib/components/ui/Skeleton.svelte';
	import CategoryDonut from '$lib/components/evaluations/CategoryDonut.svelte';
	import CategoryGradeCard from '$lib/components/evaluations/CategoryGradeCard.svelte';
	import ScoreHistoryChart, {
		type BuildHistoryEntry
	} from '$lib/components/evaluations/ScoreHistoryChart.svelte';
	import type { Site, Evaluation } from '$lib/api/types';
	import { compositeGrade } from '$lib/evaluations/grade';

	let { data }: { data: { site: Site } } = $props();
	const siteID = $derived(data.site.id);
	const buildID = $derived($page.params.buildID ?? '');

	const CATEGORIES = ['security', 'seo', 'performance', 'accessibility', 'privacy'] as const;

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

	const composite = $derived(compositeGrade(buildEvals));
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

<div class="mx-auto max-w-7xl px-6 py-8">
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
				<p class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">Composite</p>
				<div class="mt-1">
					{#if composite}
						<GradeBadge grade={composite} size="lg" />
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
		<Card padding="md" class="mb-6">
			<div class="grid grid-cols-2 gap-6 sm:grid-cols-3 lg:grid-cols-5">
				{#each CATEGORIES as cat (cat)}
					<div class="flex flex-col items-center gap-2">
						<Skeleton width="8rem" height="8rem" rounded="full" />
						<Skeleton width="3rem" height="0.7rem" />
					</div>
				{/each}
			</div>
		</Card>
		<div class="space-y-4">
			{#each Array(5) as _, i (i)}
				<Skeleton width="100%" height="120px" />
			{/each}
		</div>
	{:else if loadError}
		<Card padding="md">
			<p class="text-[13px] text-danger">{loadError}</p>
		</Card>
	{:else}
		<!-- Donut row: matches the eval index "Latest scores" panel. -->
		<section class="mb-8">
			<Card padding="md">
				<div class="flex items-baseline justify-between">
					<h2 class="text-[11px] font-mono uppercase tracking-[0.2em] text-text-muted">
						Category scores
					</h2>
					<span class="text-[11px] text-text-muted">5 categories</span>
				</div>
				<div class="mt-5 grid grid-cols-2 gap-6 sm:grid-cols-3 lg:grid-cols-5">
					{#each CATEGORIES as cat (cat)}
						<CategoryDonut category={cat} evaluation={evalByCategory[cat] ?? null} />
					{/each}
				</div>
			</Card>
		</section>

		<!-- Per-category bars: full-width, embedded donut, 2-col check grid on expand. -->
		<section class="space-y-4">
			{#each CATEGORIES as cat (cat)}
				<CategoryGradeCard
					category={cat}
					evaluation={evalByCategory[cat] ?? null}
					{siteID}
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

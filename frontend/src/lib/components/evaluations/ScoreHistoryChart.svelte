<script lang="ts" module>
	import type { Evaluation } from '$lib/api/types';

	export interface BuildHistoryEntry {
		build_id: string;
		created_at: string;
		evaluations: Evaluation[];
	}
</script>

<script lang="ts">
	let {
		builds,
		height = 220
	}: {
		builds: BuildHistoryEntry[];
		height?: number;
	} = $props();

	type CategoryKey = 'security' | 'seo' | 'performance' | 'accessibility' | 'privacy';
	const CATEGORIES: CategoryKey[] = [
		'security',
		'seo',
		'performance',
		'accessibility',
		'privacy'
	];

	// CSS variable token names per the design system.
	const COLOR_VAR: Record<CategoryKey, string> = {
		security: 'var(--color-danger, #ef4444)',
		seo: 'var(--color-accent, #6366f1)',
		performance: 'var(--color-info, #0ea5e9)',
		accessibility: 'var(--color-warning, #f59e0b)',
		privacy: 'var(--color-success, #10b981)'
	};

	const VIEW_W = 800;
	const VIEW_H = 240;
	const PAD_L = 36;
	const PAD_R = 12;
	const PAD_T = 14;
	const PAD_B = 28;

	const sorted = $derived(
		[...builds].sort(
			(a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
		)
	);

	function categoryPercent(entry: BuildHistoryEntry, cat: CategoryKey): number | null {
		const ev = entry.evaluations.find((e) => e.category === cat);
		if (!ev) return null;
		if (ev.max_score <= 0) return null;
		return Math.max(0, Math.min(100, (ev.score / ev.max_score) * 100));
	}

	interface Point {
		x: number;
		y: number;
		v: number;
	}

	function pointsFor(cat: CategoryKey): Point[] {
		const n = sorted.length;
		if (n === 0) return [];
		const innerW = VIEW_W - PAD_L - PAD_R;
		const innerH = VIEW_H - PAD_T - PAD_B;
		const step = n === 1 ? 0 : innerW / (n - 1);
		const pts: Point[] = [];
		for (let i = 0; i < n; i++) {
			const entry = sorted[i];
			if (!entry) continue;
			const v = categoryPercent(entry, cat);
			if (v === null) continue;
			const x = PAD_L + step * i;
			const y = PAD_T + innerH - (v / 100) * innerH;
			pts.push({ x, y, v });
		}
		return pts;
	}

	function smoothPath(pts: Point[]): string {
		if (pts.length === 0) return '';
		const first = pts[0];
		if (!first) return '';
		if (pts.length === 1) return `M ${first.x} ${first.y}`;
		let d = `M ${first.x} ${first.y}`;
		for (let i = 1; i < pts.length; i++) {
			const prev = pts[i - 1];
			const cur = pts[i];
			if (!prev || !cur) continue;
			const cx = (prev.x + cur.x) / 2;
			d += ` Q ${cx} ${prev.y} ${cx} ${(prev.y + cur.y) / 2}`;
			d += ` Q ${cx} ${cur.y} ${cur.x} ${cur.y}`;
		}
		return d;
	}

	let hover = $state<{ idx: number; x: number; y: number } | null>(null);

	function onPointerMove(e: PointerEvent) {
		if (sorted.length === 0) return;
		const target = e.currentTarget as SVGSVGElement;
		const rect = target.getBoundingClientRect();
		const px = ((e.clientX - rect.left) / rect.width) * VIEW_W;
		const innerW = VIEW_W - PAD_L - PAD_R;
		const step = sorted.length === 1 ? 0 : innerW / (sorted.length - 1);
		let idx = 0;
		if (step > 0) {
			idx = Math.round((px - PAD_L) / step);
			idx = Math.max(0, Math.min(sorted.length - 1, idx));
		}
		const x = PAD_L + step * idx;
		hover = { idx, x, y: 0 };
	}

	function onPointerLeave() {
		hover = null;
	}

	function formatDate(ts: string): string {
		const d = new Date(ts);
		if (Number.isNaN(d.getTime())) return '';
		return d.toLocaleDateString(undefined, {
			month: 'short',
			day: 'numeric'
		});
	}
</script>

<div class="relative w-full" style="height: {height}px;">
	{#if sorted.length === 0}
		<div
			class="flex h-full items-center justify-center rounded-lg border border-border-light bg-bg-surface text-[12px] text-text-muted"
		>
			No build history yet.
		</div>
	{:else}
		<svg
			viewBox="0 0 {VIEW_W} {VIEW_H}"
			preserveAspectRatio="none"
			class="h-full w-full"
			role="img"
			aria-label="Score history per category over recent builds"
			onpointermove={onPointerMove}
			onpointerleave={onPointerLeave}
		>
			<!-- Y axis grid lines at 0, 25, 50, 75, 100 -->
			{#each [0, 25, 50, 75, 100] as level (level)}
				{@const innerH = VIEW_H - PAD_T - PAD_B}
				{@const y = PAD_T + innerH - (level / 100) * innerH}
				<line
					x1={PAD_L}
					x2={VIEW_W - PAD_R}
					y1={y}
					y2={y}
					stroke="var(--color-border-light, rgba(0,0,0,0.08))"
					stroke-width="1"
				/>
				<text
					x={PAD_L - 6}
					y={y + 3}
					text-anchor="end"
					font-size="10"
					fill="var(--color-text-muted, #9aa0a6)"
				>
					{level}
				</text>
			{/each}

			<!-- Lines per category -->
			{#each CATEGORIES as cat (cat)}
				{@const pts = pointsFor(cat)}
				{#if pts.length > 0}
					<path
						d={smoothPath(pts)}
						fill="none"
						stroke={COLOR_VAR[cat]}
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					/>
					{#each pts as p, i (`${cat}-${i}`)}
						<circle cx={p.x} cy={p.y} r="2.5" fill={COLOR_VAR[cat]} />
					{/each}
				{/if}
			{/each}

			<!-- Hover guide line -->
			{#if hover && sorted[hover.idx]}
				{@const hoverEntry = sorted[hover.idx] as BuildHistoryEntry}
				<line
					x1={hover.x}
					x2={hover.x}
					y1={PAD_T}
					y2={VIEW_H - PAD_B}
					stroke="var(--color-text-muted, #9aa0a6)"
					stroke-width="1"
					stroke-dasharray="3 3"
					opacity="0.5"
				/>
				<text
					x={hover.x}
					y={VIEW_H - PAD_B + 16}
					text-anchor="middle"
					font-size="10"
					fill="var(--color-text-muted, #9aa0a6)"
				>
					{formatDate(hoverEntry.created_at)}
				</text>
			{/if}
		</svg>

		{#if hover && sorted[hover.idx]}
			{@const hoverEntry = sorted[hover.idx] as BuildHistoryEntry}
			<div
				class="pointer-events-none absolute top-2 right-2 rounded-md border border-border bg-bg-elevated px-2 py-1.5 text-[11px] text-text-primary shadow-sm"
			>
				<p class="font-medium">{formatDate(hoverEntry.created_at)}</p>
				<div class="mt-1 space-y-0.5">
					{#each CATEGORIES as cat (cat)}
						{@const pct = categoryPercent(hoverEntry, cat)}
						<div class="flex items-center gap-1.5">
							<span
								class="inline-block h-2 w-2 rounded-full"
								style="background: {COLOR_VAR[cat]};"
							></span>
							<span class="capitalize text-text-secondary">{cat}</span>
							<span class="ml-auto font-mono">{pct === null ? '-' : pct.toFixed(0)}</span>
						</div>
					{/each}
				</div>
			</div>
		{/if}
	{/if}

	<!-- Legend -->
	<div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-text-muted">
		{#each CATEGORIES as cat (cat)}
			<span class="inline-flex items-center gap-1.5 capitalize">
				<span
					class="inline-block h-2 w-2 rounded-full"
					style="background: {COLOR_VAR[cat]};"
				></span>
				{cat}
			</span>
		{/each}
	</div>
</div>

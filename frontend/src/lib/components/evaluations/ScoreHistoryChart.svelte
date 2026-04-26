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
		height = 240
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
	const CATEGORY_LABEL: Record<CategoryKey, string> = {
		security: 'Security',
		seo: 'SEO',
		performance: 'Performance',
		accessibility: 'Accessibility',
		privacy: 'Privacy'
	};

	// Tone-aware palette. Mirrors the donut + grade badge colours so the
	// trend chart reads in the same colour vocabulary as the rest of the
	// evaluation UI.
	const COLOR: Record<CategoryKey, string> = {
		security: '#ef4444',
		seo: '#6366f1',
		performance: '#0ea5e9',
		accessibility: '#f59e0b',
		privacy: '#10b981'
	};

	const VIEW_W = 800;
	const VIEW_H = 260;
	const PAD_L = 40;
	const PAD_R = 16;
	const PAD_T = 16;
	const PAD_B = 36;

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
			// Centre the single point when only one build is present.
			const x = n === 1 ? PAD_L + innerW / 2 : PAD_L + step * i;
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

	// Closes the line into a filled area for the gradient under each line.
	function areaPath(pts: Point[]): string {
		if (pts.length === 0) return '';
		const first = pts[0];
		const last = pts[pts.length - 1];
		if (!first || !last) return '';
		const baseY = VIEW_H - PAD_B;
		return `${smoothPath(pts)} L ${last.x} ${baseY} L ${first.x} ${baseY} Z`;
	}

	let hover = $state<{ idx: number; x: number } | null>(null);

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
		const x = sorted.length === 1 ? PAD_L + innerW / 2 : PAD_L + step * idx;
		hover = { idx, x };
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

	// Sparse x-axis labels: first, last, plus a couple in the middle when
	// there are enough builds. Avoids overlapping labels.
	const xLabels = $derived.by<{ x: number; label: string }[]>(() => {
		const n = sorted.length;
		if (n === 0) return [];
		const innerW = VIEW_W - PAD_L - PAD_R;
		if (n === 1) {
			const e = sorted[0];
			if (!e) return [];
			return [{ x: PAD_L + innerW / 2, label: formatDate(e.created_at) }];
		}
		const step = innerW / (n - 1);
		const indices: number[] = [];
		const ticks = Math.min(5, n);
		for (let i = 0; i < ticks; i++) {
			indices.push(Math.round((i * (n - 1)) / (ticks - 1)));
		}
		return indices.map((i) => {
			const e = sorted[i];
			return { x: PAD_L + step * i, label: e ? formatDate(e.created_at) : '' };
		});
	});
</script>

<div class="relative w-full" style="height: {height}px;">
	{#if sorted.length === 0}
		<div
			class="flex h-full items-center justify-center rounded-lg border border-border-light bg-bg-surface text-[12px] text-text-muted"
		>
			No build history yet.
		</div>
	{:else if sorted.length === 1}
		<div
			class="flex h-full flex-col items-center justify-center gap-2 rounded-lg border border-border-light bg-bg-surface px-6 text-center"
		>
			<p class="text-[12px] text-text-muted">
				Need at least 2 builds to plot a trend. The current scores are shown above.
			</p>
		</div>
	{:else}
		<svg
			viewBox="0 0 {VIEW_W} {VIEW_H}"
			preserveAspectRatio="xMidYMid meet"
			class="h-full w-full overflow-visible"
			role="img"
			aria-label="Score history per category over recent builds"
			onpointermove={onPointerMove}
			onpointerleave={onPointerLeave}
		>
			<defs>
				{#each CATEGORIES as cat (cat)}
					<linearGradient id="grad-{cat}" x1="0" x2="0" y1="0" y2="1">
						<stop offset="0%" stop-color={COLOR[cat]} stop-opacity="0.18" />
						<stop offset="100%" stop-color={COLOR[cat]} stop-opacity="0" />
					</linearGradient>
				{/each}
			</defs>

			<!-- Y axis grid lines at 0/50/100. Less noise than 5 levels. -->
			{#each [0, 50, 100] as level (level)}
				{@const innerH = VIEW_H - PAD_T - PAD_B}
				{@const y = PAD_T + innerH - (level / 100) * innerH}
				<line
					x1={PAD_L}
					x2={VIEW_W - PAD_R}
					y1={y}
					y2={y}
					stroke="var(--color-border-light, rgba(0,0,0,0.06))"
					stroke-width="1"
					stroke-dasharray={level === 0 ? '0' : '3 4'}
				/>
				<text
					x={PAD_L - 8}
					y={y + 4}
					text-anchor="end"
					font-size="10"
					fill="var(--color-text-muted, #9aa0a6)"
				>
					{level}
				</text>
			{/each}

			<!-- Hover guide first so lines paint on top. -->
			{#if hover}
				<line
					x1={hover.x}
					x2={hover.x}
					y1={PAD_T}
					y2={VIEW_H - PAD_B}
					stroke="var(--color-text-muted, #9aa0a6)"
					stroke-width="1"
					stroke-dasharray="3 3"
					opacity="0.4"
				/>
			{/if}

			<!-- Area fills, drawn first so lines sit on top -->
			{#each CATEGORIES as cat (cat)}
				{@const pts = pointsFor(cat)}
				{#if pts.length >= 2}
					<path d={areaPath(pts)} fill="url(#grad-{cat})" stroke="none" />
				{/if}
			{/each}

			<!-- Lines per category with bigger anchor points -->
			{#each CATEGORIES as cat (cat)}
				{@const pts = pointsFor(cat)}
				{#if pts.length > 0}
					<path
						d={smoothPath(pts)}
						fill="none"
						stroke={COLOR[cat]}
						stroke-width="2.5"
						stroke-linecap="round"
						stroke-linejoin="round"
					/>
					{#each pts as p, i (`${cat}-${i}`)}
						<circle
							cx={p.x}
							cy={p.y}
							r="3.5"
							fill="var(--color-bg-surface, #ffffff)"
							stroke={COLOR[cat]}
							stroke-width="2"
						/>
					{/each}
				{/if}
			{/each}

			<!-- X axis labels -->
			{#each xLabels as tick (tick.x)}
				<text
					x={tick.x}
					y={VIEW_H - PAD_B + 18}
					text-anchor="middle"
					font-size="10"
					fill="var(--color-text-muted, #9aa0a6)"
				>
					{tick.label}
				</text>
			{/each}

			<!-- Latest indicator -->
			{#if sorted.length > 1}
				{@const lastIdx = sorted.length - 1}
				{@const innerW = VIEW_W - PAD_L - PAD_R}
				{@const step = innerW / (sorted.length - 1)}
				<text
					x={PAD_L + step * lastIdx}
					y={PAD_T - 4}
					text-anchor="end"
					font-size="9"
					font-weight="500"
					fill="var(--color-text-muted, #9aa0a6)"
				>
					LATEST
				</text>
			{/if}
		</svg>

		{#if hover && sorted[hover.idx]}
			{@const hoverEntry = sorted[hover.idx] as BuildHistoryEntry}
			<div
				class="pointer-events-none absolute top-2 right-2 min-w-[160px] rounded-lg border border-border-light bg-bg-elevated px-3 py-2 text-[11px] text-text-primary shadow-md"
			>
				<p class="font-medium">{formatDate(hoverEntry.created_at)}</p>
				<p class="mt-0.5 font-mono text-[10px] text-text-muted truncate">
					{hoverEntry.build_id.slice(0, 12)}
				</p>
				<div class="mt-1.5 space-y-0.5">
					{#each CATEGORIES as cat (cat)}
						{@const pct = categoryPercent(hoverEntry, cat)}
						<div class="flex items-center gap-1.5">
							<span
								class="inline-block h-2 w-2 rounded-full"
								style="background: {COLOR[cat]};"
							></span>
							<span class="text-text-secondary">{CATEGORY_LABEL[cat]}</span>
							<span class="ml-auto font-mono">{pct === null ? '-' : pct.toFixed(0)}</span>
						</div>
					{/each}
				</div>
			</div>
		{/if}
	{/if}

	<!-- Legend -->
	<div class="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-text-muted">
		{#each CATEGORIES as cat (cat)}
			<span class="inline-flex items-center gap-1.5">
				<span
					class="inline-block h-2 w-2 rounded-full"
					style="background: {COLOR[cat]};"
				></span>
				{CATEGORY_LABEL[cat]}
			</span>
		{/each}
	</div>
</div>

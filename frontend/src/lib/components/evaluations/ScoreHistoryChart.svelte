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

	// Brand-aligned monochromatic palette. All five categories share the
	// brightinteraction.com teal family so the chart reads as ONE
	// coherent dataset rather than a five-colour rainbow. Differentiation
	// comes from luminance, not hue. Direct end-of-line labels carry the
	// category name so the legend doesn't have to map dot-to-line.
	const COLOR: Record<CategoryKey, string> = {
		security: '#0E7490',     // cyan-700, darkest
		seo: '#0891B2',          // cyan-600, brand primary
		performance: '#06B6D4',  // cyan-400
		accessibility: '#475569', // slate-600 (neutral foil)
		privacy: '#94A3B8'       // slate-400 (lightest neutral)
	};

	const VIEW_W = 800;
	const VIEW_H = 260;
	const PAD_L = 36;
	const PAD_R = 96; // wider right pad so end-of-line labels fit
	const PAD_T = 24;
	const PAD_B = 32;
	// Y axis floor. Most scores live in the 60-100 band; a 0-100 axis
	// crams every line into the top 20% of the chart and the variation
	// disappears. Anchor at 50 so a B/B- still looks like progress.
	const Y_MIN = 50;
	const Y_MAX = 100;

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
		const range = Y_MAX - Y_MIN;
		const pts: Point[] = [];
		for (let i = 0; i < n; i++) {
			const entry = sorted[i];
			if (!entry) continue;
			const v = categoryPercent(entry, cat);
			if (v === null) continue;
			const x = n === 1 ? PAD_L + innerW / 2 : PAD_L + step * i;
			// Clamp below to Y_MIN so a sub-50 score still shows on chart.
			const clamped = Math.max(Y_MIN, Math.min(Y_MAX, v));
			const y = PAD_T + innerH - ((clamped - Y_MIN) / range) * innerH;
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
			<!-- Y axis grid: 50/75/100. Anchored to the visible band so the
			     gridlines actually align with the data. -->
			{#each [50, 75, 100] as level (level)}
				{@const innerH = VIEW_H - PAD_T - PAD_B}
				{@const range = Y_MAX - Y_MIN}
				{@const y = PAD_T + innerH - ((level - Y_MIN) / range) * innerH}
				<line
					x1={PAD_L}
					x2={VIEW_W - PAD_R}
					y1={y}
					y2={y}
					stroke="var(--color-border-light, rgba(0,0,0,0.06))"
					stroke-width="1"
					stroke-dasharray={level === 100 ? '0' : '3 4'}
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

			<!-- Lines per category. No area fills (the rainbow of overlapping
			     gradients was the "horrible design" the user called out).
			     Anchor points only on the latest build to reduce visual
			     noise; the line itself communicates the trend. -->
			{#each CATEGORIES as cat (cat)}
				{@const pts = pointsFor(cat)}
				{#if pts.length > 0}
					<path
						d={smoothPath(pts)}
						fill="none"
						stroke={COLOR[cat]}
						stroke-width="2"
						stroke-linecap="round"
						stroke-linejoin="round"
					/>
					{@const last = pts[pts.length - 1]}
					{#if last}
						<circle
							cx={last.x}
							cy={last.y}
							r="3.5"
							fill="var(--color-bg-surface, #ffffff)"
							stroke={COLOR[cat]}
							stroke-width="2"
						/>
						<!-- End-of-line label removes the need for a coloured-dot legend.  -->
						<text
							x={last.x + 8}
							y={last.y + 4}
							text-anchor="start"
							font-size="10.5"
							font-weight="500"
							fill={COLOR[cat]}
						>
							{CATEGORY_LABEL[cat]}
						</text>
					{/if}
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

</div>

// Single source of truth for the 13-grade scale used by atomicsite
// evaluations. Mirrors internal/eval/grading.go.ScoreToGrade.
//
// Tiers + colours:
//   A+ A A-      (>= 90)  -> success    (emerald)
//   B+ B B-      (80-89)  -> success-soft (light emerald)
//   C+ C C-      (70-79)  -> warning    (amber)
//   D+ D D-      (60-69)  -> warning-strong (orange)
//   F            (< 60)   -> danger     (red)
//
// Site Inspector uses an identical scheme so cross-tool readings line up.

export type Grade =
	| 'A+' | 'A' | 'A-'
	| 'B+' | 'B' | 'B-'
	| 'C+' | 'C' | 'C-'
	| 'D+' | 'D' | 'D-'
	| 'F';

export const VALID_GRADES: Grade[] = [
	'A+', 'A', 'A-',
	'B+', 'B', 'B-',
	'C+', 'C', 'C-',
	'D+', 'D', 'D-',
	'F'
];

export const GRADE_RANK: Record<Grade, number> = {
	'A+': 0, A: 1, 'A-': 2,
	'B+': 3, B: 4, 'B-': 5,
	'C+': 6, C: 7, 'C-': 8,
	'D+': 9, D: 10, 'D-': 11,
	F: 12
};

export type Tone = 'success' | 'success-soft' | 'warning' | 'warning-strong' | 'danger' | 'muted';

export function asGrade(value: string | undefined | null): Grade | null {
	if (!value) return null;
	return (VALID_GRADES as string[]).includes(value) ? (value as Grade) : null;
}

// Compose a per-build composite as the worst category grade.
export function compositeGrade<T extends { grade: string }>(items: T[]): Grade | null {
	let worst: Grade | null = null;
	for (const it of items) {
		const g = asGrade(it.grade);
		if (!g) continue;
		if (worst === null || GRADE_RANK[g] > GRADE_RANK[worst]) worst = g;
	}
	return worst;
}

// Tone for any grade. "muted" is reserved for missing/unscored.
export function gradeTone(grade: Grade | null | undefined): Tone {
	if (!grade) return 'muted';
	const head = grade.charAt(0);
	switch (head) {
		case 'A':
			return 'success';
		case 'B':
			return 'success-soft';
		case 'C':
			return 'warning';
		case 'D':
			return 'warning-strong';
		default:
			return 'danger';
	}
}

// Tone derived from a percentage (0-100). Used by the donut ring when an
// evaluation has a numeric score but no letter grade (e.g., partial loads).
export function pctTone(pct: number): Tone {
	if (!Number.isFinite(pct)) return 'muted';
	if (pct >= 90) return 'success';
	if (pct >= 80) return 'success-soft';
	if (pct >= 70) return 'warning';
	if (pct >= 60) return 'warning-strong';
	return 'danger';
}

export function toneCSSColor(tone: Tone): string {
	switch (tone) {
		case 'success':
			return 'var(--t-success, #10b981)';
		case 'success-soft':
			return 'var(--t-success-soft, #6ee7b7)';
		case 'warning':
			return 'var(--t-warning, #f59e0b)';
		case 'warning-strong':
			return 'var(--t-warning-strong, #f97316)';
		case 'danger':
			return 'var(--t-danger, #ef4444)';
		default:
			return 'var(--t-text-muted, #94a3b8)';
	}
}

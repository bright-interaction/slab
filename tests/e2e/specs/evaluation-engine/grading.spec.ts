import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, triggerBuildAndWait } from '../../fixtures/data';
import { createPublishedPageWithBlock } from '../build-pipeline/_helpers';
import { findCategory, getEvaluations } from './_helpers';

/**
 * Mirror of internal/eval/grading.go ScoreToGrade. Assertions in this file are
 * a smoke check that the engine emits grades consistent with this table.
 */
function scoreToGrade(score: number, max: number): string {
	if (max <= 0) return 'N/A';
	const pct = (score * 100) / max;
	if (pct >= 97) return 'A+';
	if (pct >= 93) return 'A';
	if (pct >= 90) return 'A-';
	if (pct >= 87) return 'B+';
	if (pct >= 83) return 'B';
	if (pct >= 80) return 'B-';
	if (pct >= 77) return 'C+';
	if (pct >= 73) return 'C';
	if (pct >= 70) return 'C-';
	if (pct >= 67) return 'D+';
	if (pct >= 63) return 'D';
	if (pct >= 60) return 'D-';
	return 'F';
}

test.describe('evaluation-engine: grading math', () => {
	const cleanup: string[] = [];

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('every category grade matches score/max via the published table', async ({ adminApi }) => {
		test.setTimeout(240_000);
		const site = await createSite(adminApi);
		cleanup.push(site.id);
		await createPublishedPageWithBlock(adminApi, site.id, { slug: '/', title: 'Grade Home' });

		const result = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(result.status, `build_log:\n${result.build_log}`).toBe('success');

		const rows = await getEvaluations(adminApi, site.id, result.build_id);
		expect(rows.length).toBeGreaterThanOrEqual(5);

		for (const row of rows) {
			expect(row.grade).toBe(scoreToGrade(row.score, row.max_score));
		}

		// Spot-check the math: pure ratio + boundary sanity
		const sec = findCategory(rows, 'security');
		expect(sec.score).toBeLessThanOrEqual(sec.max_score);
		expect(sec.score).toBeGreaterThanOrEqual(0);

		// 90+ is A-/A/A+ family, < 60 is F. Use the security row as the probe.
		if (sec.max_score > 0) {
			const pct = (sec.score * 100) / sec.max_score;
			if (pct >= 90) expect(sec.grade.startsWith('A')).toBe(true);
			else if (pct < 60) expect(sec.grade).toBe('F');
		}
	});
});

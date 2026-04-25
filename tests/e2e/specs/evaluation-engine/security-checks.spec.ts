import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, triggerBuildAndWait } from '../../fixtures/data';
import { createPublishedPageWithBlock } from '../build-pipeline/_helpers';
import { checkNames, findCategory, getEvaluations, VALID_GRADES } from './_helpers';

test.describe('evaluation-engine: security category', () => {
	const cleanup: string[] = [];

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('security row has expected check names + numeric score + valid grade', async ({
		adminApi
	}) => {
		test.setTimeout(240_000);
		const site = await createSite(adminApi);
		cleanup.push(site.id);
		await createPublishedPageWithBlock(adminApi, site.id, { slug: '/', title: 'Sec Home' });

		const result = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(result.status, `build_log:\n${result.build_log}`).toBe('success');

		const rows = await getEvaluations(adminApi, site.id, result.build_id);
		const security = findCategory(rows, 'security');
		expect(typeof security.score).toBe('number');
		expect(security.max_score).toBeGreaterThan(0);
		expect(VALID_GRADES.has(security.grade), `grade=${security.grade}`).toBe(true);

		const names = checkNames(security);
		// Names emitted by RunSecurityChecks (internal/eval/security.go).
		for (const expected of [
			'Content-Security-Policy',
			'X-Frame-Options',
			'X-Content-Type-Options: nosniff',
			'Strict-Transport-Security',
			'Referrer-Policy'
		]) {
			expect(names, `missing ${expected} in ${names.join(',')}`).toContain(expected);
		}
	});
});

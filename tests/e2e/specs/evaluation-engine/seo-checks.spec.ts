import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, triggerBuildAndWait } from '../../fixtures/data';
import { createPublishedPageWithBlock } from '../build-pipeline/_helpers';
import { checkNames, findCategory, getEvaluations, VALID_GRADES } from './_helpers';

test.describe('evaluation-engine: seo category', () => {
	const cleanup: string[] = [];

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('seo row carries on-page + technical + files checks', async ({ adminApi }) => {
		test.setTimeout(240_000);
		const site = await createSite(adminApi);
		cleanup.push(site.id);
		await createPublishedPageWithBlock(adminApi, site.id, { slug: '/', title: 'SEO Home' });

		const result = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(result.status, `build_log:\n${result.build_log}`).toBe('success');

		const rows = await getEvaluations(adminApi, site.id, result.build_id);
		const seo = findCategory(rows, 'seo');
		expect(typeof seo.score).toBe('number');
		expect(VALID_GRADES.has(seo.grade), `grade=${seo.grade}`).toBe(true);

		const names = checkNames(seo);
		// Names from internal/eval/seo.go.
		for (const expected of [
			'Has Title',
			'Has Meta Description',
			'Has H1',
			'Canonical URL',
			'Viewport Meta',
			'robots.txt',
			'XML Sitemap',
			'llms.txt'
		]) {
			expect(names, `missing ${expected} in ${names.join(',')}`).toContain(expected);
		}
	});
});

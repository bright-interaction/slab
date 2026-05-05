import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, triggerBuildAndWait, type Site } from '../../fixtures/data';
import { createPublishedPageWithBlock } from '../build-pipeline/_helpers';

// Critique runs in the post-build goroutine alongside eval. After a
// successful build, /api/sites/{id}/evaluations should include a
// row with category="design".
test.describe('Design critique', () => {
	let site: Site;

	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('post-build hook persists a design category row', async ({ adminApi }) => {
		test.setTimeout(240_000);
		site = await createSite(adminApi, { siteType: 'b2b', structureType: 'one-pager' });
		await createPublishedPageWithBlock(adminApi, site.id, { slug: '/', title: 'Home' });

		const build = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(build.status, `build_log:\n${build.build_log}`).toBe('success');

		const res = await adminApi.get(`/api/sites/${site.id}/evaluations/${build.build_id}`);
		expect(res.ok()).toBeTruthy();
		const rows = await res.json();
		const cats = (rows as Array<{ category: string }>).map((r) => r.category);
		const design = (rows as Array<{ category: string }>).find((r) => r.category === 'design');
		expect(
			design,
			`expected category="design" row, got categories=${JSON.stringify(cats)}\nbuild_log:\n${build.build_log}`
		).toBeTruthy();
	});
});

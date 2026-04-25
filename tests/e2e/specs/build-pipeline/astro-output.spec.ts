import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, triggerBuildAndWait } from '../../fixtures/data';
import { createPublishedPageWithBlock, readDist } from './_helpers';

test.describe('build-pipeline: astro output', () => {
	const cleanup: string[] = [];

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('block content appears in rendered HTML', async ({ adminApi }) => {
		test.setTimeout(240_000);
		const site = await createSite(adminApi);
		cleanup.push(site.id);

		const { text } = await createPublishedPageWithBlock(adminApi, site.id, {
			slug: '/',
			title: 'Astro Output Home'
		});

		const result = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(result.status, `build_log:\n${result.build_log}`).toBe('success');

		// Side effect: dist HTML contains the block's text content
		const html = readDist(site.id, 'index.html');
		expect(html).toContain(text);
		// And the page <title> reflects the page title
		expect(html).toMatch(/<title>[^<]*Astro Output Home/);
	});
});

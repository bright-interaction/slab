import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, triggerBuildAndWait } from '../../fixtures/data';
import { createPublishedPageWithBlock, distDir, distExists } from './_helpers';

test.describe('build-pipeline: full build', () => {
	const cleanup: string[] = [];

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('seeded site builds successfully and writes dist/index.html', async ({ adminApi }) => {
		test.setTimeout(240_000);
		const site = await createSite(adminApi, { siteType: 'b2b', structureType: 'one-pager' });
		cleanup.push(site.id);

		await createPublishedPageWithBlock(adminApi, site.id, { slug: '/', title: 'Home' });

		const result = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(result.status, `build_log:\n${result.build_log}`).toBe('success');

		// Side effect: dist directory and index.html exist on disk
		expect(distExists(site.id, '')).toBe(true);
		expect(distExists(site.id, 'index.html')).toBe(true);

		// Side effect: build status is reflected via the admin API
		const statusRes = await adminApi.get(
			u(`/api/sites/${site.id}/builds/${result.build_id}/status`)
		);
		expect(statusRes.ok()).toBe(true);
		const body = await statusRes.json();
		expect(body.status).toBe('success');
		expect(body.dist_dir).toBe(distDir(site.id));
	});
});

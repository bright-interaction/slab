import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, triggerBuildAndWait } from '../../fixtures/data';

test.describe('build-pipeline: failure paths', () => {
	const cleanup: string[] = [];

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('a site with zero published pages fails the build with a meaningful log', async ({
		adminApi
	}) => {
		test.setTimeout(120_000);

		// Seeded sites get essential pages (Privacy/Terms/Cookies/404), but they
		// are created with status=draft, so ListPublishedPagesBySite returns 0
		// rows and the builder fails fast before bun install runs.
		const site = await createSite(adminApi);
		cleanup.push(site.id);

		const result = await triggerBuildAndWait(adminApi, site.id, 60_000);
		expect(result.status).toBe('failed');
		expect((result.error ?? '') + result.build_log).toMatch(/no published pages/i);
	});
});

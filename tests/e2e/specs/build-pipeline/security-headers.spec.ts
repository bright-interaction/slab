import { test, expect } from '../../fixtures/auth';
import { createSite, deleteSite, triggerBuildAndWait } from '../../fixtures/data';
import { createPublishedPageWithBlock, readDist } from './_helpers';

test.describe('build-pipeline: _headers (Netlify-format)', () => {
	const cleanup: string[] = [];

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('dist/_headers contains the canonical security headers', async ({ adminApi }) => {
		test.setTimeout(240_000);
		const site = await createSite(adminApi);
		cleanup.push(site.id);

		await createPublishedPageWithBlock(adminApi, site.id, { slug: '/', title: 'Headers Home' });

		const result = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(result.status, `build_log:\n${result.build_log}`).toBe('success');

		const headers = readDist(site.id, '_headers');
		expect(headers).toContain('Content-Security-Policy');
		expect(headers).toContain('X-Frame-Options');
		expect(headers).toContain('X-Content-Type-Options');
		expect(headers).toContain('Strict-Transport-Security');
		expect(headers).toContain('Referrer-Policy');
		// File starts with the global path matcher
		expect(headers).toMatch(/^[^]*\/\*\n/);
	});
});

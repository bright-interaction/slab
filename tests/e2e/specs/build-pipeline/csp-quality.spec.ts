import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, rand, triggerBuildAndWait } from '../../fixtures/data';
import { createPublishedPageWithBlock, readDist } from './_helpers';

function extractCSP(headersFile: string): string {
	const m = headersFile.match(/Content-Security-Policy:\s*([^\n]+)/);
	return m ? m[1].trim() : '';
}

test.describe('build-pipeline: CSP quality', () => {
	const cleanup: string[] = [];

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('script-src has no unsafe-inline by default', async ({ adminApi }) => {
		test.setTimeout(240_000);
		const site = await createSite(adminApi);
		cleanup.push(site.id);

		await createPublishedPageWithBlock(adminApi, site.id, { slug: '/', title: 'CSP Home' });

		const result = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(result.status, `build_log:\n${result.build_log}`).toBe('success');

		const csp = extractCSP(readDist(site.id, '_headers'));
		expect(csp).not.toBe('');

		// Find script-src directive specifically
		const scriptSrcMatch = csp.match(/script-src\s+([^;]+)/);
		expect(scriptSrcMatch, `csp:\n${csp}`).not.toBeNull();
		const scriptSrc = scriptSrcMatch![1];
		// Default builder output uses 'self' only - no unsafe-inline (and if it
		// ever appears, it must be paired with a nonce-* token).
		const hasUnsafeInline = scriptSrc.includes("'unsafe-inline'");
		const hasNonce = /nonce-/.test(scriptSrc);
		expect(hasUnsafeInline && !hasNonce, `script-src=${scriptSrc}`).toBe(false);
	});

	test('allowed-scripts entries are reflected in script-src', async ({ adminApi }) => {
		test.setTimeout(240_000);
		const site = await createSite(adminApi);
		cleanup.push(site.id);

		await createPublishedPageWithBlock(adminApi, site.id, { slug: '/', title: 'CSP Allow' });

		const domain = `${rand('cdn')}.example.test`;
		const addRes = await adminApi.post(u(`/api/sites/${site.id}/allowed-scripts`), {
			data: { domain }
		});
		expect(addRes.ok(), await addRes.text()).toBe(true);

		const result = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(result.status, `build_log:\n${result.build_log}`).toBe('success');

		const csp = extractCSP(readDist(site.id, '_headers'));
		expect(csp).toContain(domain);
	});
});

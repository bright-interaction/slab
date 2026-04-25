import { test, expect, u } from '../../fixtures/auth';
import {
	createSite,
	deleteSite,
	getStubCalls,
	resetStub,
	triggerBuildAndWait,
	type Site
} from '../../fixtures/data';

// internal/handlers/build.go calls cookieproof.EnsureDomain at build time
// when:
//   - cfg.CookieProofAdminToken is set (it is, in webServer.env)
//   - cfg.CookieProofAPIBase is set (points at our stub)
//   - the site has a non-empty cookieproof_domain or domain
//   - the site setting analytics.cookieproof_enabled is "1"/"true"/"yes"/"on"
//
// EnsureDomain then PUT /api/config + POST /api/settings/domains. We assert
// the stub recorded both.

test.describe('integrations: CookieProof provision-on-build', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi, { siteType: 'b2b', structureType: 'one-pager' });
		cleanup.push(site.id);

		// Flip cookieproof_enabled on so build.go runs the provisioning step.
		const res = await adminApi.put(u(`/api/sites/${site.id}/settings`), {
			data: { category: 'analytics', key: 'cookieproof_enabled', value: '1' }
		});
		expect(res.ok()).toBe(true);

		// Provisioning runs regardless of build success, but we still want a
		// published page so the rest of the build pipeline doesn't error early.
		// Not strictly required for this spec, but mirrors a real flow.
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('first build POSTs to CookieProof /api/config and /api/settings/domains', async ({
		adminApi
	}) => {
		await resetStub('cookieproof');

		const build = await triggerBuildAndWait(adminApi, site.id, 180_000);
		expect(['success', 'failed']).toContain(build.status);
		// Build outcome is irrelevant to provisioning; provisioning runs even
		// if the bun build later crashes. Assert on the captured calls.

		const calls = (await getStubCalls('cookieproof')) as Array<{
			method: string;
			path: string;
			body: unknown;
		}>;
		const putConfig = calls.find((c) => c.method === 'PUT' && c.path === '/api/config');
		expect(putConfig, `no PUT /api/config; calls=${JSON.stringify(calls)}`).toBeTruthy();
		const cfgBody = putConfig!.body as Record<string, unknown>;
		expect(typeof cfgBody.domain).toBe('string');
		expect(String(cfgBody.domain)).toBe(site.domain);

		const postOrigin = calls.find(
			(c) => c.method === 'POST' && c.path === '/api/settings/domains'
		);
		expect(postOrigin, 'no POST /api/settings/domains').toBeTruthy();
		const originBody = postOrigin!.body as Record<string, unknown>;
		expect(String(originBody.origin ?? '')).toBe(`https://${site.domain}`);
	});

	test('CookieProof disabled site does not call the stub on build', async ({ adminApi }) => {
		const off = await createSite(adminApi);
		cleanup.push(off.id);
		// analytics.cookieproof_enabled stays unset / falsey.
		await resetStub('cookieproof');
		const build = await triggerBuildAndWait(adminApi, off.id, 180_000);
		expect(['success', 'failed']).toContain(build.status);

		const calls = (await getStubCalls('cookieproof')) as unknown[];
		expect(calls).toHaveLength(0);
	});
});

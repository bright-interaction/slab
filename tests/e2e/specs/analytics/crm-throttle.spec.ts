import { request } from '@playwright/test';
import { test, expect, u } from '../../fixtures/auth';
import {
	createSite,
	deleteSite,
	getStubCalls,
	resetStub,
	type Site
} from '../../fixtures/data';

// internal/crmsync/throttle.go suppresses repeated *non-identified* events
// for the same (site, visitor). Identified events (the ones /t/consent emits)
// bypass throttling by design, so two consents back-to-back must BOTH land
// in the CRM stub.
//
// /t/pageview itself does not currently call the CRM client, so the only
// outbound channel we can observe from e2e is the consent path. We assert
// the bypass-on-identified property here.

async function postConsent(ctx: Awaited<ReturnType<typeof request.newContext>>, siteId: string) {
	return ctx.post(u('/t/consent'), {
		data: {
			siteId,
			page: '/',
			consent: {
				version: 1,
				timestamp: Math.floor(Date.now() / 1000),
				method: 'accept-all',
				categories: { necessary: true, analytics: true }
			}
		}
	});
}

test.describe('analytics: identified events bypass throttler', () => {
	let site: Site;
	const cleanup: string[] = [];

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('three rapid consents all reach CRM (identified events skip throttle)', async () => {
		await resetStub('crm');
		const ctx = await request.newContext({
			extraHTTPHeaders: { 'Content-Type': 'application/json' }
		});
		try {
			const r1 = await postConsent(ctx, site.id);
			const r2 = await postConsent(ctx, site.id);
			const r3 = await postConsent(ctx, site.id);
			expect([r1.status(), r2.status(), r3.status()]).toEqual([204, 204, 204]);
		} finally {
			await ctx.dispose();
		}

		// Wait for SendAsync goroutines.
		let calls: unknown[] = [];
		const deadline = Date.now() + 5_000;
		while (Date.now() < deadline) {
			calls = (await getStubCalls('crm')) as unknown[];
			if (calls.length >= 3) break;
			await new Promise((r) => setTimeout(r, 100));
		}
		expect(calls.length).toBeGreaterThanOrEqual(3);
	});
});

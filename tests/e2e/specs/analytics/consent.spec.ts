import { request } from '@playwright/test';
import { test, expect, u } from '../../fixtures/auth';
import {
	createSite,
	deleteSite,
	getStubCalls,
	resetStub,
	type Site
} from '../../fixtures/data';

// /t/consent flow: the in-page CookieProof relay POSTs a payload after the
// visitor opts in. Server upserts a visit_session, identifies it, and fires
// an "identified" event to BrightCRM (our stub). 204 on success.

test.describe('analytics: consent updates session + emits identified event', () => {
	let site: Site;
	const cleanup: string[] = [];

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('analytics opt-in returns 204 and lands an identified event in the CRM stub', async () => {
		await resetStub('crm');

		const ctx = await request.newContext({
			extraHTTPHeaders: {
				'Content-Type': 'application/json',
				'User-Agent': 'e2e-consent/1.0',
				'Accept-Language': 'en-US'
			}
		});
		try {
			const res = await ctx.post(u('/t/consent'), {
				data: {
					siteId: site.id,
					sessionId: 'sess-e2e',
					page: 'https://e2e.example.test/',
					referrer: '',
					consent: {
						version: 1,
						timestamp: Math.floor(Date.now() / 1000),
						method: 'accept-all',
						gpc: false,
						categories: { necessary: true, analytics: true, functional: true }
					}
				}
			});
			expect(res.status()).toBe(204);
		} finally {
			await ctx.dispose();
		}

		// SendAsync goroutine fires after the response; wait briefly.
		let calls: Array<{ method: string; path: string; body: unknown }> = [];
		const deadline = Date.now() + 5_000;
		while (Date.now() < deadline) {
			calls = (await getStubCalls('crm')) as typeof calls;
			if (calls.length >= 1) break;
			await new Promise((r) => setTimeout(r, 100));
		}
		expect(calls.length, 'expected at least one CRM webhook call').toBeGreaterThanOrEqual(1);
		const first = calls[0];
		expect(first.method).toBe('POST');
		const body = first.body as Record<string, unknown>;
		expect(body.event).toBe('identified');
		expect(body.site_id).toBe(site.id);
		expect(typeof body.visitor_id).toBe('string');
	});

	test('reject-all consent does NOT fire a CRM event', async () => {
		await resetStub('crm');
		const ctx = await request.newContext({
			extraHTTPHeaders: { 'Content-Type': 'application/json' }
		});
		try {
			const res = await ctx.post(u('/t/consent'), {
				data: {
					siteId: site.id,
					page: '/',
					consent: {
						version: 1,
						timestamp: Math.floor(Date.now() / 1000),
						method: 'reject-all',
						gpc: false,
						categories: { necessary: true, analytics: false }
					}
				}
			});
			expect(res.status()).toBe(204);
		} finally {
			await ctx.dispose();
		}
		await new Promise((r) => setTimeout(r, 500));
		const calls = (await getStubCalls('crm')) as unknown[];
		expect(calls).toHaveLength(0);
	});

	test('unknown siteId returns 404', async () => {
		const ctx = await request.newContext({
			extraHTTPHeaders: { 'Content-Type': 'application/json' }
		});
		try {
			const res = await ctx.post(u('/t/consent'), {
				data: {
					siteId: 'aaaaaaaaaaaaaaaaaaaaaaaa',
					consent: {
						version: 1,
						timestamp: 0,
						method: 'accept-all',
						categories: { necessary: true, analytics: true }
					}
				}
			});
			expect(res.status()).toBe(404);
		} finally {
			await ctx.dispose();
		}
	});
});

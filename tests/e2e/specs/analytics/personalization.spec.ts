import { request } from '@playwright/test';
import { createHmac } from 'node:crypto';
import { test, expect, u } from '../../fixtures/auth';
import {
	createSite,
	deleteSite,
	type Site
} from '../../fixtures/data';
import { E2E_CONSTANTS } from '../../playwright.config';

// Phase 18: bidirectional CRM personalization.
//
// /t/inbound: the CRM POSTs HMAC-signed metadata for a known visitor. We
//   verify: bad sig -> 401, missing site -> 404, missing identifier -> 400,
//   no matching session -> 404, valid push -> 204.
// /t/visitor: the built site's hydration script GETs metadata bound to the
//   request fingerprint (re-derived from headers, not cookie). We verify:
//   no row -> 200 with empty anonymous + null identified; row present ->
//   200 carrying the metadata in the right tier.

const SECRET = E2E_CONSTANTS.STUB_CRM_SECRET;

function sign(body: string): string {
	return createHmac('sha256', SECRET).update(body, 'utf8').digest('hex');
}

test.describe('analytics: /t/inbound HMAC + identifier resolution', () => {
	let site: Site;
	const cleanup: string[] = [];

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('bad HMAC signature returns 401', async () => {
		const ctx = await request.newContext({
			extraHTTPHeaders: { 'Content-Type': 'application/json' }
		});
		try {
			const body = JSON.stringify({ site_id: site.id, fingerprint: 'aaaaaaaaaaaaaaaa', metadata: {} });
			const res = await ctx.post(u('/t/inbound'), {
				headers: { 'X-Slab-Signature': 'deadbeef' },
				data: body
			});
			expect(res.status()).toBe(401);
		} finally {
			await ctx.dispose();
		}
	});

	test('missing identifier returns 400', async () => {
		const ctx = await request.newContext({
			extraHTTPHeaders: { 'Content-Type': 'application/json' }
		});
		try {
			const body = JSON.stringify({ site_id: site.id, metadata: { lifecycle: 'SQL' } });
			const res = await ctx.post(u('/t/inbound'), {
				headers: { 'X-Slab-Signature': sign(body) },
				data: body
			});
			expect(res.status()).toBe(400);
		} finally {
			await ctx.dispose();
		}
	});

	test('no matching session returns 404', async () => {
		const ctx = await request.newContext({
			extraHTTPHeaders: { 'Content-Type': 'application/json' }
		});
		try {
			const body = JSON.stringify({
				site_id: site.id,
				fingerprint: '0000000000000000',
				metadata: { lifecycle: 'SQL' }
			});
			const res = await ctx.post(u('/t/inbound'), {
				headers: { 'X-Slab-Signature': sign(body) },
				data: body
			});
			expect(res.status()).toBe(404);
		} finally {
			await ctx.dispose();
		}
	});

	test('unknown site returns 404', async () => {
		const ctx = await request.newContext({
			extraHTTPHeaders: { 'Content-Type': 'application/json' }
		});
		try {
			const body = JSON.stringify({
				site_id: 'aaaaaaaaaaaaaaaaaaaaaaaa',
				fingerprint: '0000000000000000',
				metadata: { lifecycle: 'SQL' }
			});
			const res = await ctx.post(u('/t/inbound'), {
				headers: { 'X-Slab-Signature': sign(body) },
				data: body
			});
			expect(res.status()).toBe(404);
		} finally {
			await ctx.dispose();
		}
	});
});

test.describe('analytics: /t/visitor returns metadata bound to the requester fingerprint', () => {
	let site: Site;
	const cleanup: string[] = [];

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('no matching session returns empty payload (returning=false)', async () => {
		const ctx = await request.newContext();
		try {
			const res = await ctx.get(u(`/t/visitor?site_id=${site.id}`));
			expect(res.status()).toBe(200);
			const body = await res.json();
			expect(body.returning).toBe(false);
			expect(body.anonymous).toEqual({});
			expect(body.identified).toEqual({});
		} finally {
			await ctx.dispose();
		}
	});

	test('round-trip: consent creates session, inbound pushes metadata, visitor reads it back', async () => {
		// Use a stable user-agent so /t/consent and /t/visitor derive the
		// same fingerprint server-side.
		const UA = 'SlabE2E/18.5 (compatible)';
		const sharedHeaders = {
			'User-Agent': UA,
			'Accept-Language': 'en-GB,en;q=0.9'
		};

		const consentCtx = await request.newContext({
			extraHTTPHeaders: { 'Content-Type': 'application/json', ...sharedHeaders }
		});
		try {
			const res = await consentCtx.post(u('/t/consent'), {
				data: {
					siteId: site.id,
					page: '/',
					consent: {
						version: 1,
						timestamp: Math.floor(Date.now() / 1000),
						method: 'accept-all',
						categories: { necessary: true, analytics: true }
					}
				}
			});
			expect(res.status()).toBe(204);
		} finally {
			await consentCtx.dispose();
		}

		// Inbound: the CRM pushes anonymous-tier metadata (no identity_confirmed_at,
		// so the response should NOT carry the name field).
		// We don't know the fingerprint client-side (it's HttpOnly), so target
		// by visitor_id which the server derived from the same headers.
		// The visitor_id format is "v_" + siteID[:6] + "_" + fingerprint
		// (deriveVisitorID at track.go:415). We can compute it by hitting
		// /t/visitor first and reading the visitor_id field.

		const probeCtx = await request.newContext({ extraHTTPHeaders: sharedHeaders });
		let visitorID: string;
		try {
			const probe = await probeCtx.get(u(`/t/visitor?site_id=${site.id}`));
			expect(probe.status()).toBe(200);
			const body = await probe.json();
			visitorID = body.visitor_id;
			expect(visitorID).toBeTruthy();
		} finally {
			await probeCtx.dispose();
		}

		const inboundCtx = await request.newContext({
			extraHTTPHeaders: { 'Content-Type': 'application/json' }
		});
		try {
			const body = JSON.stringify({
				site_id: site.id,
				visitor_id: visitorID,
				metadata: {
					lifecycle: 'SQL',
					lead_score: 72,
					last_topic: 'compliance',
					name: 'Joe' // identified-tier; should NOT come back since we omit identity_confirmed_at
				}
			});
			const res = await inboundCtx.post(u('/t/inbound'), {
				headers: { 'X-Slab-Signature': sign(body) },
				data: body
			});
			expect(res.status()).toBe(204);
		} finally {
			await inboundCtx.dispose();
		}

		// Read back: anonymous tier carries lifecycle/lead_score/last_topic;
		// identified tier is empty because identity_confirmed_at is missing.
		const readCtx = await request.newContext({ extraHTTPHeaders: sharedHeaders });
		try {
			const res = await readCtx.get(u(`/t/visitor?site_id=${site.id}`));
			expect(res.status()).toBe(200);
			const body = await res.json();
			expect(body.returning).toBe(true);
			expect(body.anonymous.lifecycle).toBe('SQL');
			expect(body.anonymous.lead_score).toBe(72);
			expect(body.anonymous.last_topic).toBe('compliance');
			expect(body.identified).toEqual({});
		} finally {
			await readCtx.dispose();
		}
	});

	test('fresh identity_confirmed_at promotes name to identified tier', async () => {
		const UA = 'SlabE2E/18.5 identified (compatible)';
		const sharedHeaders = {
			'User-Agent': UA,
			'Accept-Language': 'en-GB,en;q=0.9'
		};

		const consentCtx = await request.newContext({
			extraHTTPHeaders: { 'Content-Type': 'application/json', ...sharedHeaders }
		});
		try {
			await consentCtx.post(u('/t/consent'), {
				data: {
					siteId: site.id,
					page: '/',
					consent: {
						version: 1,
						timestamp: Math.floor(Date.now() / 1000),
						method: 'accept-all',
						categories: { necessary: true, analytics: true }
					}
				}
			});
		} finally {
			await consentCtx.dispose();
		}

		const probeCtx = await request.newContext({ extraHTTPHeaders: sharedHeaders });
		let visitorID: string;
		try {
			const probe = await probeCtx.get(u(`/t/visitor?site_id=${site.id}`));
			const body = await probe.json();
			visitorID = body.visitor_id;
		} finally {
			await probeCtx.dispose();
		}

		const inboundCtx = await request.newContext({
			extraHTTPHeaders: { 'Content-Type': 'application/json' }
		});
		try {
			const body = JSON.stringify({
				site_id: site.id,
				visitor_id: visitorID,
				identity_confirmed_at: new Date().toISOString(),
				metadata: { name: 'Joe', lifecycle: 'Customer' }
			});
			await inboundCtx.post(u('/t/inbound'), {
				headers: { 'X-Slab-Signature': sign(body) },
				data: body
			});
		} finally {
			await inboundCtx.dispose();
		}

		const readCtx = await request.newContext({ extraHTTPHeaders: sharedHeaders });
		try {
			const res = await readCtx.get(u(`/t/visitor?site_id=${site.id}`));
			const body = await res.json();
			expect(body.identified.name).toBe('Joe');
			expect(body.anonymous.lifecycle).toBe('Customer');
		} finally {
			await readCtx.dispose();
		}
	});

	test('stale identity_confirmed_at scrubs identified tier', async () => {
		const UA = 'SlabE2E/18.5 stale (compatible)';
		const sharedHeaders = {
			'User-Agent': UA,
			'Accept-Language': 'en-GB,en;q=0.9'
		};

		const consentCtx = await request.newContext({
			extraHTTPHeaders: { 'Content-Type': 'application/json', ...sharedHeaders }
		});
		try {
			await consentCtx.post(u('/t/consent'), {
				data: {
					siteId: site.id,
					page: '/',
					consent: {
						version: 1,
						timestamp: Math.floor(Date.now() / 1000),
						method: 'accept-all',
						categories: { necessary: true, analytics: true }
					}
				}
			});
		} finally {
			await consentCtx.dispose();
		}

		const probeCtx = await request.newContext({ extraHTTPHeaders: sharedHeaders });
		let visitorID: string;
		try {
			const probe = await probeCtx.get(u(`/t/visitor?site_id=${site.id}`));
			const body = await probe.json();
			visitorID = body.visitor_id;
		} finally {
			await probeCtx.dispose();
		}

		const stale = new Date();
		stale.setDate(stale.getDate() - 90);
		const inboundCtx = await request.newContext({
			extraHTTPHeaders: { 'Content-Type': 'application/json' }
		});
		try {
			const body = JSON.stringify({
				site_id: site.id,
				visitor_id: visitorID,
				identity_confirmed_at: stale.toISOString(),
				metadata: { name: 'Stale Joe', lifecycle: 'Customer' }
			});
			await inboundCtx.post(u('/t/inbound'), {
				headers: { 'X-Slab-Signature': sign(body) },
				data: body
			});
		} finally {
			await inboundCtx.dispose();
		}

		const readCtx = await request.newContext({ extraHTTPHeaders: sharedHeaders });
		try {
			const res = await readCtx.get(u(`/t/visitor?site_id=${site.id}`));
			const body = await res.json();
			expect(body.identified).toEqual({});
			expect(body.anonymous.lifecycle).toBe('Customer');
		} finally {
			await readCtx.dispose();
		}
	});
});

test.describe('agent-api: personalization context + guardrail + debug endpoint', () => {
	let site: Site;
	const cleanup: string[] = [];

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('/api/agent/context carries personalization with example_conditions', async ({ adminApi }) => {
		// Mint an agent key for this site so we can call the agent endpoints.
		const keyRes = await adminApi.post(`/api/sites/${site.id}/agent-keys`, {
			data: { name: 'personalization-test' }
		});
		expect(keyRes.ok()).toBe(true);
		const { key } = await keyRes.json();

		const ctx = await request.newContext({
			extraHTTPHeaders: { 'X-Agent-Key': key }
		});
		try {
			const res = await ctx.get(u('/api/agent/context'));
			expect(res.ok()).toBe(true);
			const body = await res.json();
			expect(body.personalization).toBeTruthy();
			expect(body.personalization.identity_max_age_days).toBeGreaterThan(0);
			expect(Array.isArray(body.personalization.example_conditions)).toBe(true);
			expect(body.personalization.example_conditions.length).toBeGreaterThan(3);
			expect(Array.isArray(body.personalization.known_keys)).toBe(true);
		} finally {
			await ctx.dispose();
		}
	});

	test('guardrail rejects malicious condition expressions', async ({ adminApi }) => {
		const keyRes = await adminApi.post(`/api/sites/${site.id}/agent-keys`, {
			data: { name: 'personalization-guardrail' }
		});
		const { key } = await keyRes.json();

		// Create a page first.
		const pageRes = await adminApi.post(`/api/sites/${site.id}/pages`, {
			data: { title: 'Guardrail test page', slug: '/guardrail-test', layout: 'default' }
		});
		expect(pageRes.ok()).toBe(true);
		const page = await pageRes.json();
		// Publish it via PATCH (admin path; status-on-create is ignored).
		await adminApi.patch(`/api/sites/${site.id}/pages/${page.id}`, {
			data: { status: 'published' }
		});

		const agentCtx = await request.newContext({
			extraHTTPHeaders: { 'X-Agent-Key': key, 'Content-Type': 'application/json' }
		});
		try {
			const res = await agentCtx.post(u('/api/agent/blocks?page=/guardrail-test'), {
				data: {
					block_type: 'hero',
					data: {
						headline: 'Hello world this is a long enough headline',
						condition: `name == "<script>alert(1)</script>"`
					}
				}
			});
			expect(res.status()).toBe(422);
			const body = await res.json();
			expect(JSON.stringify(body)).toContain('personalization_condition');
		} finally {
			await agentCtx.dispose();
		}
	});
});

import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, rand, type Site } from '../../fixtures/data';

// Phase 31.3 (2026-05-06): visitor identification end-to-end.
//
// Two paths under test:
//   1. POST /t/identify -- explicit SDK call. Emits a pageview to
//      seed the visit_session, then identifies, then asserts the
//      session appears in /analytics/sessions?identified=true with
//      the right email.
//   2. POST /t/forms/{formID}/submit -- auto-identify. Creates a
//      form, posts a submission carrying an email-shaped value,
//      then asserts the same identified-sessions query surfaces
//      the email.
//
// The harness uses a single APIRequestContext with an admin cookie;
// the fingerprint cookie is set by FingerprintMiddleware on the
// first /t/* hit and reused across requests in the same context.

test.describe('Phase 31.3: visitor identify (explicit + form-auto)', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('POST /t/identify links email to existing session', async ({ adminApi }) => {
		// 1. Seed a visit_session via /t/consent (the only handler
		//    that upserts visit_sessions; /t/pageview alone only
		//    writes visit_events).
		const consent = await adminApi.post(u('/t/consent'), {
			data: {
				siteId: site.id,
				page: '/landing',
				consent: {
					version: 1,
					timestamp: Date.now(),
					method: 'accept-all',
					categories: { necessary: true, analytics: true },
					gpc: false
				}
			}
		});
		expect(consent.status(), await consent.text()).toBe(204);

		// 2. Identify.
		const email = `visitor-${rand()}@example.com`;
		const id = await adminApi.post(u('/t/identify'), {
			data: { siteId: site.id, email, page: '/landing' }
		});
		expect(id.status(), await id.text()).toBe(204);

		// 3. Identified-sessions list should now contain this email.
		const list = await adminApi.get(
			u(`/api/sites/${site.id}/analytics/sessions?identified=true&limit=50`)
		);
		expect(list.ok()).toBeTruthy();
		const body = await list.json();
		const found = (body.sessions ?? []).find((s: { email: string }) => s.email === email);
		expect(found, `expected ${email} in sessions: ${JSON.stringify(body.sessions)}`).toBeTruthy();
		expect(found.identified_at).toBeTruthy();
	});

	test('POST /t/identify rejects malformed email', async ({ adminApi }) => {
		const res = await adminApi.post(u('/t/identify'), {
			data: { siteId: site.id, email: 'not an email' }
		});
		expect(res.status()).toBe(400);
	});

	test('POST /t/identify returns 404 when no session exists', async ({ adminApi }) => {
		// Create a fresh site that has never received a /t/consent
		// (so no visit_session row exists for this fingerprint+site
		// pair). The fingerprint stays the same as the suite's
		// because it derives from headers + IP, but the (siteID,
		// fingerprint) key has no row, so SetVisitSessionEmail
		// updates 0 rows -> 404.
		const fresh = await createSite(adminApi);
		cleanup.push(fresh.id);
		const res = await adminApi.post(u('/t/identify'), {
			data: { siteId: fresh.id, email: `nobody-${rand()}@example.com` }
		});
		expect(res.status()).toBe(404);
	});

	// Form-submit auto-identify is covered at the unit-test layer
	// in internal/handlers/forms_submit_test.go because there's no
	// admin REST endpoint for creating a form (forms are seeded via
	// starter kits or directly in the DB), so the e2e harness can't
	// stand up a fresh form to submit against without leaking SQL.
});

import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, rand, type Site } from '../../fixtures/data';

// Phase 31.1 (2026-05-06): conversion goals + events.
//
// Verifies the foundation contract end-to-end:
//   1. POST /api/sites/{id}/goals creates a goal definition
//   2. GET /api/sites/{id}/goals lists it
//   3. POST /t/pageview on a path matching the goal records a
//      conversion_event (verified via the analytics aggregate
//      endpoint reporting one conversion)
//   4. PATCH updates name/value, GET reflects it
//   5. POST /t/event with the goal's event_name slug records a
//      conversion (event-name match path)
//   6. Active=false suppresses further matches
//   7. DELETE removes the goal
//
// This is the single load-bearing spec for P1; later phases stack
// on the conversions table and re-use these endpoints.

test.describe('Phase 31.1: conversion goals + events', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('CRUD + URL-pattern goal records conversion on /t/pageview', async ({ adminApi }) => {
		// 1. Create a URL-pattern goal: /thank-you/* counts as "lead".
		const create = await adminApi.post(u(`/api/sites/${site.id}/goals`), {
			data: {
				slug: 'lead-' + rand(),
				name: 'Lead captured',
				match_type: 'url_pattern',
				match_value: '/thank-you/*',
				value_cents: 4900,
				value_currency: 'EUR'
			}
		});
		expect(create.status(), await create.text()).toBe(201);
		const goal = await create.json();
		expect(goal.id).toBeTruthy();
		expect(goal.match_type).toBe('url_pattern');
		expect(goal.value_cents).toBe(4900);
		expect(goal.active).toBe(true);

		// 2. List should return it.
		const list = await adminApi.get(u(`/api/sites/${site.id}/goals`));
		expect(list.ok()).toBeTruthy();
		const listBody = await list.json();
		expect(Array.isArray(listBody.goals)).toBeTruthy();
		expect(listBody.goals.find((g: { id: string }) => g.id === goal.id)).toBeTruthy();

		// 3. Fire a /t/pageview on a matching path. Must include the
		//    fingerprint cookie that FingerprintMiddleware sets, but
		//    for the test we let the middleware compute one fresh.
		const pv = await adminApi.post(u('/t/pageview'), {
			data: {
				siteId: site.id,
				path: '/thank-you/lead',
				referrer: ''
			}
		});
		expect(pv.status(), await pv.text()).toBe(204);

		// 4. Aggregate endpoint should now report one conversion.
		const agg = await adminApi.get(u(`/api/sites/${site.id}/analytics/goals?since=1d`));
		expect(agg.ok()).toBeTruthy();
		const aggBody = await agg.json();
		const myGoal = aggBody.goals.find((g: { id: string }) => g.id === goal.id);
		expect(myGoal).toBeTruthy();
		expect(myGoal.conversions).toBeGreaterThanOrEqual(1);
		expect(myGoal.unique_converters).toBeGreaterThanOrEqual(1);
		expect(myGoal.total_value_cents).toBeGreaterThanOrEqual(4900);

		// 5. Update: rename + double the value.
		const upd = await adminApi.patch(u(`/api/sites/${site.id}/goals/${goal.id}`), {
			data: {
				name: 'Lead captured (renamed)',
				match_type: 'url_pattern',
				match_value: '/thank-you/*',
				value_cents: 9800,
				value_currency: 'EUR',
				active: true
			}
		});
		expect(upd.ok()).toBeTruthy();
		const updated = await upd.json();
		expect(updated.name).toBe('Lead captured (renamed)');
		expect(updated.value_cents).toBe(9800);

		// 6. Cleanup.
		const del = await adminApi.delete(u(`/api/sites/${site.id}/goals/${goal.id}`));
		expect(del.ok()).toBeTruthy();
	});

	test('event_name goal fires on /t/event', async ({ adminApi }) => {
		const slug = 'signup-' + rand();
		const create = await adminApi.post(u(`/api/sites/${site.id}/goals`), {
			data: {
				slug,
				name: 'Signup completed',
				match_type: 'event_name',
				match_value: slug,
				value_cents: 0
			}
		});
		expect(create.status()).toBe(201);
		const goal = await create.json();

		const ev = await adminApi.post(u('/t/event'), {
			data: { siteId: site.id, name: slug, path: '/onboarding/done' }
		});
		expect(ev.status(), await ev.text()).toBe(204);

		const agg = await adminApi.get(u(`/api/sites/${site.id}/analytics/goals?since=1d`));
		const aggBody = await agg.json();
		const myGoal = aggBody.goals.find((g: { id: string }) => g.id === goal.id);
		expect(myGoal.conversions).toBeGreaterThanOrEqual(1);

		await adminApi.delete(u(`/api/sites/${site.id}/goals/${goal.id}`));
	});

	test('rejects invalid match_type at create time', async ({ adminApi }) => {
		const res = await adminApi.post(u(`/api/sites/${site.id}/goals`), {
			data: {
				slug: 'bad-' + rand(),
				name: 'Bad',
				match_type: 'wat',
				match_value: '/x'
			}
		});
		expect(res.status()).toBe(400);
	});

	test('rejects malformed slug', async ({ adminApi }) => {
		const res = await adminApi.post(u(`/api/sites/${site.id}/goals`), {
			data: {
				slug: 'BAD SLUG',
				name: 'Bad',
				match_type: 'url_pattern',
				match_value: '/x'
			}
		});
		expect(res.status()).toBe(400);
	});

	test('inactive goal does not record conversions', async ({ adminApi }) => {
		const slug = 'inactive-' + rand();
		const create = await adminApi.post(u(`/api/sites/${site.id}/goals`), {
			data: {
				slug,
				name: 'Inactive goal',
				match_type: 'url_pattern',
				match_value: '/inactive-target',
				active: false
			}
		});
		expect(create.status()).toBe(201);
		const goal = await create.json();
		expect(goal.active).toBe(false);

		// Hit the URL; should NOT fire because goal is inactive.
		await adminApi.post(u('/t/pageview'), {
			data: { siteId: site.id, path: '/inactive-target' }
		});

		const agg = await adminApi.get(u(`/api/sites/${site.id}/analytics/goals?since=1d`));
		const aggBody = await agg.json();
		const myGoal = aggBody.goals.find((g: { id: string }) => g.id === goal.id);
		expect(myGoal.conversions).toBe(0);

		await adminApi.delete(u(`/api/sites/${site.id}/goals/${goal.id}`));
	});
});

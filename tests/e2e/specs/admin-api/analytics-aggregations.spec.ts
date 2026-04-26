// Coverage for the new analytics aggregation endpoints introduced in
// Phase 12.5: /analytics/overview, /analytics/sessions,
// /analytics/conversion-paths, /analytics/tracked-fields. The previous
// frontend was calling these against a backend that only shipped
// /visit-events, so the dashboard always fell back to its empty state.
import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../fixtures/data';

test.describe('admin-api: analytics aggregations', () => {
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
	});
	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('GET /analytics/overview returns full empty shape on a fresh site', async ({ adminApi }) => {
		const res = await adminApi.get(u(`/api/sites/${site.id}/analytics/overview?since=7d`));
		expect(res.ok()).toBe(true);
		const body = await res.json();

		// Expected fields present and typed correctly even on an empty site.
		expect(body.since).toBe('7d');
		expect(body.bucket_unit).toBe('day');
		expect(body.visits).toBe(0);
		expect(body.unique_visitors).toBe(0);
		expect(body.identified_count).toBe(0);
		expect(body.live_visitors).toBe(0);
		expect(Array.isArray(body.top_pages)).toBe(true);
		expect(Array.isArray(body.top_referers)).toBe(true);
		expect(Array.isArray(body.top_browsers)).toBe(true);
		expect(Array.isArray(body.top_os)).toBe(true);
		expect(Array.isArray(body.top_devices)).toBe(true);
		expect(Array.isArray(body.top_countries)).toBe(true);
		expect(Array.isArray(body.top_languages)).toBe(true);
		expect(Array.isArray(body.top_utm_sources)).toBe(true);
		expect(Array.isArray(body.top_utm_campaigns)).toBe(true);
		expect(Array.isArray(body.time_series)).toBe(true);
	});

	test('GET /analytics/overview honours since=1d (hourly buckets)', async ({ adminApi }) => {
		const res = await adminApi.get(u(`/api/sites/${site.id}/analytics/overview?since=1d`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.bucket_unit).toBe('hour');
	});

	test('GET /analytics/sessions without identified=true returns empty list', async ({ adminApi }) => {
		const res = await adminApi.get(u(`/api/sites/${site.id}/analytics/sessions`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.sessions).toEqual([]);
	});

	test('GET /analytics/sessions?identified=true returns shape', async ({ adminApi }) => {
		const res = await adminApi.get(
			u(`/api/sites/${site.id}/analytics/sessions?identified=true&limit=10`)
		);
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(Array.isArray(body.sessions)).toBe(true);
	});

	test('GET /analytics/conversion-paths returns shape', async ({ adminApi }) => {
		const res = await adminApi.get(u(`/api/sites/${site.id}/analytics/conversion-paths?limit=5`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(Array.isArray(body.paths)).toBe(true);
	});

	test('GET /analytics/tracked-fields lists tracked + not-tracked fields', async ({ adminApi }) => {
		const res = await adminApi.get(u(`/api/sites/${site.id}/analytics/tracked-fields`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(Array.isArray(body.tracked)).toBe(true);
		expect(Array.isArray(body.not_tracked)).toBe(true);
		expect(body.tracked.length).toBeGreaterThan(8);
		expect(body.not_tracked.length).toBeGreaterThan(2);
		// Each tracked entry must have the four keys the UI expects.
		for (const t of body.tracked) {
			expect(typeof t.field).toBe('string');
			expect(typeof t.stored).toBe('string');
			expect(typeof t.source).toBe('string');
			expect(typeof t.purpose).toBe('string');
		}
		// IP must be in not_tracked: that's the user-facing privacy promise.
		const ipNotTracked = body.not_tracked.find((f: { field: string }) => f.field === 'ip');
		expect(ipNotTracked).toBeTruthy();
	});

	test('all four endpoints require admin auth', async ({ request }) => {
		const paths = [
			`/api/sites/${site.id}/analytics/overview`,
			`/api/sites/${site.id}/analytics/sessions`,
			`/api/sites/${site.id}/analytics/conversion-paths`,
			`/api/sites/${site.id}/analytics/tracked-fields`
		];
		for (const p of paths) {
			const res = await request.get(u(p));
			expect(res.status(), `${p} should require auth`).toBe(401);
		}
	});
});

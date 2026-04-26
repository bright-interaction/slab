// Coverage for the inline engagement beacon (Phase 12.6). The browser side
// is a vanilla <script> baked into Layout.astro; this spec exercises the
// public POST /t/engagement receiver directly and verifies the data lands
// in the analytics overview's engagement summary.
import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../fixtures/data';

test.describe('analytics: engagement beacon', () => {
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
	});
	test.afterAll(async ({ adminApi }) => {
		if (site) await deleteSite(adminApi, site.id);
	});

	test('POST /t/engagement records a row, /analytics/overview surfaces aggregates', async ({
		request,
		adminApi
	}) => {
		// Pre-flight: overview engagement is empty on a fresh site.
		const before = await adminApi.get(u(`/api/sites/${site.id}/analytics/overview?since=7d`));
		expect(before.ok()).toBe(true);
		const beforeBody = await before.json();
		expect(beforeBody.engagement.samples).toBe(0);

		// POST a beacon. No auth required (public receiver).
		const beaconRes = await request.post(u('/t/engagement'), {
			data: {
				siteId: site.id,
				path: '/pricing',
				screenW: 1920,
				screenH: 1080,
				viewportW: 1440,
				viewportH: 900,
				prefersDark: true,
				prefersReducedMotion: false,
				timeOnPageMs: 90_000,
				maxScrollPct: 75
			}
		});
		expect(beaconRes.status()).toBe(204);

		// Overview now reflects the sample.
		const after = await adminApi.get(u(`/api/sites/${site.id}/analytics/overview?since=7d`));
		expect(after.ok()).toBe(true);
		const afterBody = await after.json();
		expect(afterBody.engagement.samples).toBe(1);
		expect(afterBody.engagement.avg_time_on_page_ms).toBe(90_000);
		expect(afterBody.engagement.avg_scroll_pct).toBe(75);
		expect(afterBody.engagement.dark_pct).toBe(100); // 1/1 prefers dark
		expect(afterBody.engagement.reduced_motion_pct).toBe(0);
		expect(afterBody.engagement.by_path.length).toBeGreaterThan(0);
		expect(afterBody.engagement.by_path[0].path).toBe('/pricing');
		expect(afterBody.engagement.viewport_buckets.length).toBeGreaterThan(0);
		expect(afterBody.engagement.viewport_buckets[0].name).toBe('1400px');
	});

	test('rejects unknown siteId with 404', async ({ request }) => {
		// Atomicsite site IDs are 24 hex chars (12 bytes); use a valid-shape
		// ID that doesn't exist in the DB so we hit the 404 branch instead
		// of the 400 (invalid format) branch.
		const res = await request.post(u('/t/engagement'), {
			data: { siteId: 'aaaaaaaaaaaaaaaaaaaaaaaa', path: '/' }
		});
		expect(res.status()).toBe(404);
	});

	test('clamps absurd values (timeOnPageMs > 6h, maxScrollPct > 100)', async ({
		request,
		adminApi
	}) => {
		const res = await request.post(u('/t/engagement'), {
			data: {
				siteId: site.id,
				path: '/clamp-test',
				timeOnPageMs: 999_999_999,
				maxScrollPct: 250
			}
		});
		expect(res.status()).toBe(204);

		const overview = await adminApi.get(u(`/api/sites/${site.id}/analytics/overview?since=7d`));
		const body = await overview.json();
		const clampRow = body.engagement.by_path.find((r: { path: string }) => r.path === '/clamp-test');
		expect(clampRow).toBeTruthy();
		expect(clampRow.avg_time_on_page_ms).toBeLessThanOrEqual(6 * 60 * 60 * 1000);
		expect(clampRow.avg_scroll_pct).toBeLessThanOrEqual(100);
	});

	test('tracked-fields panel includes the beacon-derived fields', async ({ adminApi }) => {
		const res = await adminApi.get(u(`/api/sites/${site.id}/analytics/tracked-fields`));
		const body = await res.json();
		const fields = body.tracked.map((t: { field: string }) => t.field);
		expect(fields).toContain('screen size');
		expect(fields).toContain('viewport size');
		expect(fields).toContain('prefers_dark');
		expect(fields).toContain('time_on_page_ms');
		expect(fields).toContain('max_scroll_pct');
	});
});

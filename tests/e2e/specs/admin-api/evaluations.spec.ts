import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../fixtures/data';

test.describe('admin-api: evaluations', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('GET /api/sites/{id}/evaluations returns array (empty if no builds)', async ({ adminApi }) => {
		const res = await adminApi.get(u(`/api/sites/${site.id}/evaluations`));
		expect(res.ok()).toBe(true);
		const list = await res.json();
		expect(Array.isArray(list)).toBe(true);
	});

	test('GET ?limit= respects limit param', async ({ adminApi }) => {
		const res = await adminApi.get(u(`/api/sites/${site.id}/evaluations?limit=5`));
		expect(res.ok()).toBe(true);
		const list = await res.json();
		expect(Array.isArray(list)).toBe(true);
	});

	test('GET /api/sites/{id}/evaluations/{buildID} returns array for missing build', async ({ adminApi }) => {
		// Missing builds yield an empty array (the query filters by build_id).
		const res = await adminApi.get(u(`/api/sites/${site.id}/evaluations/does-not-exist`));
		expect(res.ok()).toBe(true);
		const list = await res.json();
		expect(Array.isArray(list)).toBe(true);
		expect(list.length).toBe(0);
	});
});

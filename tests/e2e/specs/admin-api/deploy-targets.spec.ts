import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, seedDeployTarget, type Site } from '../../fixtures/data';

test.describe('admin-api: deploy-targets CRUD', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('POST creates a local deploy target', async ({ adminApi }) => {
		const t = await seedDeployTarget(adminApi, site.id, 'local');
		expect(t.id).toBeTruthy();
		expect(t.kind).toBe('local');
		expect(t.is_default).toBe(true);
	});

	test('POST creates an rsync deploy target', async ({ adminApi }) => {
		const t = await seedDeployTarget(adminApi, site.id, 'rsync');
		expect(t.id).toBeTruthy();
		expect(t.kind).toBe('rsync');
	});

	test('POST creates a dockyard deploy target', async ({ adminApi }) => {
		// Validator requires https + UUID server_id; override the factory defaults.
		const t = await seedDeployTarget(adminApi, site.id, 'dockyard', {
			dockyard_url: 'https://dockyard.example.test',
			server_id: '11111111-1111-1111-1111-111111111111'
		});
		expect(t.id).toBeTruthy();
		expect(t.kind).toBe('dockyard');
	});

	test('GET lists targets wrapped in {targets:[...]}', async ({ adminApi }) => {
		const t = await seedDeployTarget(adminApi, site.id, 'local');
		const res = await adminApi.get(u(`/api/sites/${site.id}/deploy-targets`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(Array.isArray(body.targets)).toBe(true);
		expect(body.targets.find((x: { id: string }) => x.id === t.id)).toBeTruthy();
	});

	test('GET /{id} returns one target', async ({ adminApi }) => {
		const t = await seedDeployTarget(adminApi, site.id, 'local');
		const res = await adminApi.get(u(`/api/sites/${site.id}/deploy-targets/${t.id}`));
		expect(res.ok()).toBe(true);
		const got = await res.json();
		expect(got.id).toBe(t.id);
	});

	test('POST /{id}/default flips is_default', async ({ adminApi }) => {
		const a = await seedDeployTarget(adminApi, site.id, 'local');
		const b = await seedDeployTarget(adminApi, site.id, 'local');
		// b is now default (is_default: true on create flipped a off).
		const res = await adminApi.post(u(`/api/sites/${site.id}/deploy-targets/${a.id}/default`));
		expect(res.ok()).toBe(true);
		const updated = await res.json();
		expect(updated.id).toBe(a.id);
		expect(updated.is_default).toBe(true);

		const list = await adminApi.get(u(`/api/sites/${site.id}/deploy-targets`));
		const body = await list.json();
		const bAfter = body.targets.find((x: { id: string }) => x.id === b.id);
		expect(bAfter.is_default).toBe(false);
	});

	test('DELETE removes a target', async ({ adminApi }) => {
		const t = await seedDeployTarget(adminApi, site.id, 'local');
		const res = await adminApi.delete(u(`/api/sites/${site.id}/deploy-targets/${t.id}`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.status).toBe('deleted');
	});
});

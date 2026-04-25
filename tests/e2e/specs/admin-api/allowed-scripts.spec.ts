import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, rand, type Site } from '../../fixtures/data';

test.describe('admin-api: allowed-scripts CRUD', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('POST creates an allowed script', async ({ adminApi }) => {
		const domain = `${rand('cdn')}.example.com`;
		const res = await adminApi.post(u(`/api/sites/${site.id}/allowed-scripts`), {
			data: { domain, purpose: 'analytics' }
		});
		expect(res.status()).toBe(201);
		const body = await res.json();
		expect(body.id).toBeTruthy();
		expect(body.domain).toBe(domain);
	});

	test('POST rejects domain with disallowed characters', async ({ adminApi }) => {
		const res = await adminApi.post(u(`/api/sites/${site.id}/allowed-scripts`), {
			data: { domain: "evil'.example.com", purpose: 'xss' }
		});
		expect(res.status()).toBe(400);
	});

	test('GET lists allowed scripts', async ({ adminApi }) => {
		const create = await adminApi.post(u(`/api/sites/${site.id}/allowed-scripts`), {
			data: { domain: `${rand('list')}.example.com`, purpose: 'cdn' }
		});
		const created = await create.json();
		const res = await adminApi.get(u(`/api/sites/${site.id}/allowed-scripts`));
		expect(res.ok()).toBe(true);
		const list = await res.json();
		expect(Array.isArray(list)).toBe(true);
		expect(list.find((s: { id: string }) => s.id === created.id)).toBeTruthy();
	});

	test('PATCH toggles is_active', async ({ adminApi }) => {
		const create = await adminApi.post(u(`/api/sites/${site.id}/allowed-scripts`), {
			data: { domain: `${rand('toggle')}.example.com`, purpose: 'cdn' }
		});
		const created = await create.json();
		const res = await adminApi.patch(
			u(`/api/sites/${site.id}/allowed-scripts/${created.id}`),
			{ data: { is_active: 0 } }
		);
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.status).toBe('updated');

		const list = await adminApi.get(u(`/api/sites/${site.id}/allowed-scripts`));
		const items = await list.json();
		const found = items.find((s: { id: string }) => s.id === created.id);
		expect(found.is_active).toBe(0);
	});

	test('DELETE removes the script', async ({ adminApi }) => {
		const create = await adminApi.post(u(`/api/sites/${site.id}/allowed-scripts`), {
			data: { domain: `${rand('rm')}.example.com`, purpose: 'cdn' }
		});
		const created = await create.json();
		const res = await adminApi.delete(
			u(`/api/sites/${site.id}/allowed-scripts/${created.id}`)
		);
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.status).toBe('deleted');
	});
});

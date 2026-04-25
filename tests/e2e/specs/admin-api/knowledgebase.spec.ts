import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, rand, type Site } from '../../fixtures/data';

test.describe('admin-api: knowledgebase CRUD', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	async function createEntry(adminApi: import('@playwright/test').APIRequestContext, overrides: Record<string, unknown> = {}) {
		const res = await adminApi.post(u(`/api/sites/${site.id}/knowledgebase`), {
			data: {
				category: 'brand',
				title: rand('kb'),
				content: 'This is a knowledge base entry.',
				sort_order: 0,
				...overrides
			}
		});
		expect(res.status()).toBe(201);
		return res.json();
	}

	test('POST creates an entry', async ({ adminApi }) => {
		const e = await createEntry(adminApi);
		expect(e.id).toBeTruthy();
		expect(e.category).toBe('brand');
		expect(e.is_active).toBe(1);
	});

	test('GET lists entries (seeded with defaults)', async ({ adminApi }) => {
		const e = await createEntry(adminApi);
		const res = await adminApi.get(u(`/api/sites/${site.id}/knowledgebase`));
		expect(res.ok()).toBe(true);
		const list = await res.json();
		expect(Array.isArray(list)).toBe(true);
		expect(list.find((x: { id: string }) => x.id === e.id)).toBeTruthy();
	});

	test('GET /{id} returns one entry', async ({ adminApi }) => {
		const e = await createEntry(adminApi);
		const res = await adminApi.get(u(`/api/sites/${site.id}/knowledgebase/${e.id}`));
		expect(res.ok()).toBe(true);
		const got = await res.json();
		expect(got.id).toBe(e.id);
	});

	test('PATCH updates content', async ({ adminApi }) => {
		const e = await createEntry(adminApi);
		const res = await adminApi.patch(u(`/api/sites/${site.id}/knowledgebase/${e.id}`), {
			data: { content: 'Updated content' }
		});
		expect(res.ok()).toBe(true);
		const updated = await res.json();
		expect(updated.content).toBe('Updated content');
	});

	test('PATCH toggles is_active', async ({ adminApi }) => {
		const e = await createEntry(adminApi);
		const res = await adminApi.patch(u(`/api/sites/${site.id}/knowledgebase/${e.id}`), {
			data: { is_active: 0 }
		});
		expect(res.ok()).toBe(true);
		const updated = await res.json();
		expect(updated.is_active).toBe(0);
	});

	test('DELETE removes an entry', async ({ adminApi }) => {
		const e = await createEntry(adminApi);
		const res = await adminApi.delete(u(`/api/sites/${site.id}/knowledgebase/${e.id}`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.status).toBe('deleted');
	});
});

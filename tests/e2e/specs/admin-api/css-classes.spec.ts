import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, rand, type Site } from '../../fixtures/data';

test.describe('admin-api: css-classes CRUD', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('POST creates a CSS class with sort_order', async ({ adminApi }) => {
		const res = await adminApi.post(u(`/api/sites/${site.id}/css-classes`), {
			data: {
				name: rand('cls'),
				category: 'utility',
				css: '.foo { color: red; }',
				usage_note: 'red text',
				sort_order: 5
			}
		});
		expect(res.status()).toBe(201);
		const cc = await res.json();
		expect(cc.id).toBeTruthy();
		expect(cc.sort_order).toBe(5);
	});

	test('GET lists CSS classes', async ({ adminApi }) => {
		const create = await adminApi.post(u(`/api/sites/${site.id}/css-classes`), {
			data: { name: rand('cls'), css: '.bar { color: blue; }' }
		});
		const cc = await create.json();
		const res = await adminApi.get(u(`/api/sites/${site.id}/css-classes`));
		expect(res.ok()).toBe(true);
		const list = await res.json();
		expect(Array.isArray(list)).toBe(true);
		expect(list.find((x: { id: string }) => x.id === cc.id)).toBeTruthy();
	});

	test('GET /{id} returns a single class', async ({ adminApi }) => {
		const create = await adminApi.post(u(`/api/sites/${site.id}/css-classes`), {
			data: { name: rand('cls'), css: '.baz { color: green; }' }
		});
		const cc = await create.json();
		const res = await adminApi.get(u(`/api/sites/${site.id}/css-classes/${cc.id}`));
		expect(res.ok()).toBe(true);
		const got = await res.json();
		expect(got.id).toBe(cc.id);
	});

	test('PATCH updates css and sort_order', async ({ adminApi }) => {
		const create = await adminApi.post(u(`/api/sites/${site.id}/css-classes`), {
			data: { name: rand('cls'), css: '.x { color: red; }', sort_order: 1 }
		});
		const cc = await create.json();
		const res = await adminApi.patch(u(`/api/sites/${site.id}/css-classes/${cc.id}`), {
			data: { css: '.x { color: black; }', sort_order: 99 }
		});
		expect(res.ok()).toBe(true);
		const updated = await res.json();
		expect(updated.sort_order).toBe(99);
		expect(updated.css).toContain('black');
	});

	test('DELETE removes a class', async ({ adminApi }) => {
		const create = await adminApi.post(u(`/api/sites/${site.id}/css-classes`), {
			data: { name: rand('cls'), css: '.gone { display: none; }' }
		});
		const cc = await create.json();
		const res = await adminApi.delete(u(`/api/sites/${site.id}/css-classes/${cc.id}`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.status).toBe('deleted');
	});
});

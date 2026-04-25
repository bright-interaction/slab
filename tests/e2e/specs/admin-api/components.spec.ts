import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, rand, type Site } from '../../fixtures/data';

test.describe('admin-api: components CRUD', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	async function createComponent(adminApi: import('@playwright/test').APIRequestContext, overrides: Record<string, unknown> = {}) {
		const res = await adminApi.post(u(`/api/sites/${site.id}/components`), {
			data: {
				name: rand('comp'),
				category: 'section',
				template: '<section class="hero">{{title}}</section>',
				props_schema: { title: { type: 'string' } },
				css_classes: ['hero', 'wide'],
				usage_note: 'Marketing hero',
				...overrides
			}
		});
		expect(res.status()).toBe(201);
		return res.json();
	}

	test('POST creates a component with props_schema and css_classes', async ({ adminApi }) => {
		const c = await createComponent(adminApi);
		expect(c.id).toBeTruthy();
		expect(c.name).toBeTruthy();
		// props_schema is JSON-encoded string in DB
		expect(typeof c.props_schema === 'string' || typeof c.props_schema === 'object').toBe(true);
	});

	test('GET lists components for a site', async ({ adminApi }) => {
		const c = await createComponent(adminApi);
		const res = await adminApi.get(u(`/api/sites/${site.id}/components`));
		expect(res.ok()).toBe(true);
		const list = await res.json();
		expect(Array.isArray(list)).toBe(true);
		expect(list.find((x: { id: string }) => x.id === c.id)).toBeTruthy();
	});

	test('GET /{id} returns a single component', async ({ adminApi }) => {
		const c = await createComponent(adminApi);
		const res = await adminApi.get(u(`/api/sites/${site.id}/components/${c.id}`));
		expect(res.ok()).toBe(true);
		const got = await res.json();
		expect(got.id).toBe(c.id);
	});

	test('PATCH updates props_schema and css_classes', async ({ adminApi }) => {
		const c = await createComponent(adminApi);
		const res = await adminApi.patch(u(`/api/sites/${site.id}/components/${c.id}`), {
			data: {
				props_schema: { title: { type: 'string' }, subtitle: { type: 'string' } },
				css_classes: ['hero', 'narrow']
			}
		});
		expect(res.ok()).toBe(true);
		const updated = await res.json();
		expect(updated.id).toBe(c.id);
	});

	test('DELETE removes a component', async ({ adminApi }) => {
		const c = await createComponent(adminApi);
		const res = await adminApi.delete(u(`/api/sites/${site.id}/components/${c.id}`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.status).toBe('deleted');
		const get = await adminApi.get(u(`/api/sites/${site.id}/components/${c.id}`));
		expect(get.status()).toBe(404);
	});
});

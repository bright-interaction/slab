import { test, expect, u } from '../../fixtures/auth';
import { createSite, createPage, deleteSite, rand, type Site } from '../../fixtures/data';

test.describe('admin-api: pages CRUD', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('POST /api/sites/{id}/pages creates a page', async ({ adminApi }) => {
		const slug = rand('p');
		const res = await adminApi.post(u(`/api/sites/${site.id}/pages`), {
			data: { title: 'Hello E2E', slug, layout: 'default' }
		});
		expect(res.status()).toBe(201);
		const page = await res.json();
		expect(page.id).toBeTruthy();
		expect(page.slug).toBe(slug);
		expect(page.title).toBe('Hello E2E');
		expect(page.layout).toBe('default');
	});

	test('GET /api/sites/{id}/pages lists site pages', async ({ adminApi }) => {
		const created = await createPage(adminApi, site.id);
		const res = await adminApi.get(u(`/api/sites/${site.id}/pages`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(Array.isArray(body.pages)).toBe(true);
		expect(body.pages.find((p: { id: string }) => p.id === created.id)).toBeTruthy();
	});

	test('GET /api/sites/{id}/pages/{pageID} returns one page', async ({ adminApi }) => {
		const created = await createPage(adminApi, site.id);
		const res = await adminApi.get(u(`/api/sites/${site.id}/pages/${created.id}`));
		expect(res.ok()).toBe(true);
		const page = await res.json();
		expect(page.id).toBe(created.id);
	});

	test('PATCH updates status published/draft', async ({ adminApi }) => {
		const created = await createPage(adminApi, site.id);
		const res = await adminApi.patch(u(`/api/sites/${site.id}/pages/${created.id}`), {
			data: { status: 'published' }
		});
		expect(res.ok()).toBe(true);
		const page = await res.json();
		expect(page.status).toBe('published');

		const res2 = await adminApi.patch(u(`/api/sites/${site.id}/pages/${created.id}`), {
			data: { status: 'draft' }
		});
		expect(res2.ok()).toBe(true);
		const page2 = await res2.json();
		expect(page2.status).toBe('draft');
	});

	test('PATCH updates layout value', async ({ adminApi }) => {
		const created = await createPage(adminApi, site.id);
		const res = await adminApi.patch(u(`/api/sites/${site.id}/pages/${created.id}`), {
			data: { layout: 'wide' }
		});
		expect(res.ok()).toBe(true);
		const page = await res.json();
		expect(page.layout).toBe('wide');
	});

	test('PATCH toggles no_index', async ({ adminApi }) => {
		const created = await createPage(adminApi, site.id);
		const res = await adminApi.patch(u(`/api/sites/${site.id}/pages/${created.id}`), {
			data: { no_index: true }
		});
		expect(res.ok()).toBe(true);
		const page = await res.json();
		expect(page.no_index).toBe(1);
	});

	test('PATCH sets canonical_url', async ({ adminApi }) => {
		const created = await createPage(adminApi, site.id);
		const canonical = 'https://example.test/canonical-path';
		const res = await adminApi.patch(u(`/api/sites/${site.id}/pages/${created.id}`), {
			data: { canonical_url: canonical }
		});
		expect(res.ok()).toBe(true);
		const page = await res.json();
		expect(page.canonical_url).toBe(canonical);
	});

	test('POST /api/sites/{id}/pages/reorder updates sort order', async ({ adminApi }) => {
		const a = await createPage(adminApi, site.id);
		const b = await createPage(adminApi, site.id);
		const res = await adminApi.post(u(`/api/sites/${site.id}/pages/reorder`), {
			data: {
				order: [
					{ id: a.id, sort_order: 99 },
					{ id: b.id, sort_order: 100 }
				]
			}
		});
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.status).toBe('ok');
		const get = await adminApi.get(u(`/api/sites/${site.id}/pages/${a.id}`));
		const page = await get.json();
		expect(page.sort_order).toBe(99);
	});

	test('DELETE removes a page', async ({ adminApi }) => {
		const created = await createPage(adminApi, site.id);
		const res = await adminApi.delete(u(`/api/sites/${site.id}/pages/${created.id}`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.status).toBe('deleted');
		const get = await adminApi.get(u(`/api/sites/${site.id}/pages/${created.id}`));
		expect(get.status()).toBe(404);
	});
});

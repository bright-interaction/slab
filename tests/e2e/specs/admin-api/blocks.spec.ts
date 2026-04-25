import { test, expect, u } from '../../fixtures/auth';
import { createBlock, createPage, createSite, deleteSite, type Site, type Page_ } from '../../fixtures/data';

test.describe('admin-api: blocks CRUD', () => {
	const cleanup: string[] = [];
	let site: Site;
	let page: Page_;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
		page = await createPage(adminApi, site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('POST creates a block', async ({ adminApi }) => {
		const res = await adminApi.post(u(`/api/sites/${site.id}/pages/${page.id}/blocks`), {
			data: { block_type: 'text', data: { content: 'hi' }, sort_order: 0 }
		});
		expect(res.status()).toBe(201);
		const block = await res.json();
		expect(block.id).toBeTruthy();
		expect(block.block_type).toBe('text');
	});

	test('GET lists blocks on a page', async ({ adminApi }) => {
		const created = await createBlock(adminApi, site.id, page.id);
		const res = await adminApi.get(u(`/api/sites/${site.id}/pages/${page.id}/blocks`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(Array.isArray(body.blocks)).toBe(true);
		expect(body.blocks.find((b: { id: string }) => b.id === created.id)).toBeTruthy();
	});

	test('PATCH updates a block', async ({ adminApi }) => {
		const created = await createBlock(adminApi, site.id, page.id);
		const res = await adminApi.patch(
			u(`/api/sites/${site.id}/pages/${page.id}/blocks/${created.id}`),
			{ data: { block_type: 'hero' } }
		);
		expect(res.ok()).toBe(true);
		const block = await res.json();
		expect(block.block_type).toBe('hero');
	});

	test('PATCH toggles is_visible', async ({ adminApi }) => {
		const created = await createBlock(adminApi, site.id, page.id);
		const res = await adminApi.patch(
			u(`/api/sites/${site.id}/pages/${page.id}/blocks/${created.id}`),
			{ data: { is_visible: false } }
		);
		expect(res.ok()).toBe(true);
		const block = await res.json();
		expect(block.is_visible).toBe(0);
	});

	test('POST reorder updates sort_order', async ({ adminApi }) => {
		const a = await createBlock(adminApi, site.id, page.id, { sort_order: 0 });
		const b = await createBlock(adminApi, site.id, page.id, { sort_order: 1 });
		const res = await adminApi.post(
			u(`/api/sites/${site.id}/pages/${page.id}/blocks/reorder`),
			{
				data: {
					order: [
						{ id: a.id, sort_order: 50 },
						{ id: b.id, sort_order: 51 }
					]
				}
			}
		);
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.status).toBe('ok');
	});

	test('PUT bulk save replaces all blocks', async ({ adminApi }) => {
		// fresh page so this test is isolated
		const fresh = await createPage(adminApi, site.id);
		const res = await adminApi.put(u(`/api/sites/${site.id}/pages/${fresh.id}/blocks/bulk`), {
			data: {
				blocks: [
					{ block_type: 'hero', sort_order: 0, data: { title: 'a' }, style: {}, is_visible: true },
					{ block_type: 'text', sort_order: 1, data: { content: 'b' }, style: {}, is_visible: true }
				]
			}
		});
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(Array.isArray(body.blocks)).toBe(true);
		expect(body.blocks.length).toBe(2);
		expect(body.blocks[0].block_type).toBe('hero');
	});

	test('DELETE removes a block', async ({ adminApi }) => {
		const created = await createBlock(adminApi, site.id, page.id);
		const res = await adminApi.delete(
			u(`/api/sites/${site.id}/pages/${page.id}/blocks/${created.id}`)
		);
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.status).toBe('deleted');
	});
});

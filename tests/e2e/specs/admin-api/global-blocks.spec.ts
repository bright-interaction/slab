import { test, expect, u } from '../../fixtures/auth';
import {
	createBlock,
	createPage,
	createSite,
	deleteSite,
	type Site,
	type Page_
} from '../../fixtures/data';

test.describe('admin-api: global blocks CRUD + per-page suppression', () => {
	const cleanup: string[] = [];
	let site: Site;
	let otherSite: Site;
	let page: Page_;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
		otherSite = await createSite(adminApi);
		cleanup.push(otherSite.id);
		page = await createPage(adminApi, site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('GET lists global blocks (initial seed creates a header + footer)', async ({
		adminApi
	}) => {
		const res = await adminApi.get(u(`/api/sites/${site.id}/global-blocks`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(Array.isArray(body.global_blocks)).toBe(true);
	});

	test('POST creates a global block in the header slot', async ({ adminApi }) => {
		const res = await adminApi.post(u(`/api/sites/${site.id}/global-blocks`), {
			data: {
				name: 'Custom landing nav',
				slot: 'header',
				block_type: 'nav',
				data: { links: [{ label: 'Pricing', href: '/pricing' }] }
			}
		});
		expect(res.status()).toBe(201);
		const block = await res.json();
		expect(block.id).toBeTruthy();
		expect(block.slot).toBe('header');
		expect(block.name).toBe('Custom landing nav');
		expect(block.is_active).toBe(1);
	});

	test('POST rejects an invalid slot', async ({ adminApi }) => {
		const res = await adminApi.post(u(`/api/sites/${site.id}/global-blocks`), {
			data: {
				name: 'Sidebar',
				slot: 'sidebar',
				block_type: 'nav',
				data: {}
			}
		});
		expect(res.status()).toBe(400);
	});

	test('POST requires a name', async ({ adminApi }) => {
		const res = await adminApi.post(u(`/api/sites/${site.id}/global-blocks`), {
			data: { slot: 'footer', block_type: 'footer', data: {} }
		});
		expect(res.status()).toBe(400);
	});

	test('PATCH updates name + data + is_active without re-creating', async ({ adminApi }) => {
		const create = await adminApi.post(u(`/api/sites/${site.id}/global-blocks`), {
			data: { name: 'Hello', slot: 'footer', block_type: 'footer', data: {} }
		});
		const created = await create.json();

		const res = await adminApi.patch(
			u(`/api/sites/${site.id}/global-blocks/${created.id}`),
			{ data: { name: 'Bye', is_active: false, data: { copyright: '2026' } } }
		);
		expect(res.ok()).toBe(true);
		const updated = await res.json();
		expect(updated.name).toBe('Bye');
		expect(updated.is_active).toBe(0);
		expect(updated.data_json).toContain('copyright');
	});

	test('PATCH rejects an invalid slot value', async ({ adminApi }) => {
		const create = await adminApi.post(u(`/api/sites/${site.id}/global-blocks`), {
			data: { name: 'Slot test', slot: 'header', block_type: 'nav', data: {} }
		});
		const created = await create.json();
		const res = await adminApi.patch(
			u(`/api/sites/${site.id}/global-blocks/${created.id}`),
			{ data: { slot: 'sidebar' } }
		);
		expect(res.status()).toBe(400);
	});

	test('GET cross-site returns 404', async ({ adminApi }) => {
		const create = await adminApi.post(u(`/api/sites/${site.id}/global-blocks`), {
			data: { name: 'Stay here', slot: 'header', block_type: 'nav', data: {} }
		});
		const created = await create.json();
		const res = await adminApi.get(
			u(`/api/sites/${otherSite.id}/global-blocks/${created.id}`)
		);
		expect(res.status()).toBe(404);
	});

	test('DELETE removes the block', async ({ adminApi }) => {
		const create = await adminApi.post(u(`/api/sites/${site.id}/global-blocks`), {
			data: { name: 'Temp', slot: 'footer', block_type: 'footer', data: {} }
		});
		const created = await create.json();

		const del = await adminApi.delete(
			u(`/api/sites/${site.id}/global-blocks/${created.id}`)
		);
		expect(del.ok()).toBe(true);

		const after = await adminApi.get(u(`/api/sites/${site.id}/global-blocks/${created.id}`));
		expect(after.status()).toBe(404);
	});

	test('Page preview emits hideGlobalBlocks={true} when the page sets the flag', async ({
		adminApi
	}) => {
		const fresh = await createPage(adminApi, site.id, { title: 'Landing', slug: 'landing' });
		await createBlock(adminApi, site.id, fresh.id, {
			block_type: 'hero',
			data: { headline: 'Sign up' }
		});
		// Default: prop should NOT be present.
		let res = await adminApi.get(u(`/api/sites/${site.id}/pages/${fresh.id}/preview`));
		let body = await res.json();
		expect(body.astro).not.toContain('hideGlobalBlocks');

		// Flip the suppression toggle via the same PATCH the SEO panel hits.
		const patch = await adminApi.patch(u(`/api/sites/${site.id}/pages/${fresh.id}`), {
			data: { hide_global_blocks: true }
		});
		expect(patch.ok()).toBe(true);

		res = await adminApi.get(u(`/api/sites/${site.id}/pages/${fresh.id}/preview`));
		body = await res.json();
		expect(body.astro).toContain('hideGlobalBlocks={true}');

		// Flip back off.
		await adminApi.patch(u(`/api/sites/${site.id}/pages/${fresh.id}`), {
			data: { hide_global_blocks: false }
		});
		res = await adminApi.get(u(`/api/sites/${site.id}/pages/${fresh.id}/preview`));
		body = await res.json();
		expect(body.astro).not.toContain('hideGlobalBlocks');
	});
});

test.describe('admin-api: agent context advertises preview endpoints', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('agent context returns block_schemas + endpoints + editing_modes', async ({
		adminApi
	}) => {
		// Generate an agent key for this site, then use it to fetch /api/agent/context.
		const keyRes = await adminApi.post(u(`/api/sites/${site.id}/agent-keys`), {
			data: { name: 'context-probe', capabilities: ['read', 'write'] }
		});
		expect(keyRes.ok()).toBe(true);
		const keyBody = await keyRes.json();
		const agentKey = keyBody.key;
		expect(agentKey).toBeTruthy();

		const ctx = await adminApi.get(u('/api/agent/context'), {
			headers: { 'X-Agent-Key': agentKey }
		});
		expect(ctx.ok()).toBe(true);
		const body = await ctx.json();

		// Endpoints catalogue
		expect(body.endpoints).toBeTruthy();
		expect(Array.isArray(body.endpoints.preview)).toBe(true);
		const previewPaths = body.endpoints.preview.map((e: { path: string }) => e.path);
		expect(previewPaths.some((p: string) => p.includes('/blocks/{blockID}/preview'))).toBe(true);
		expect(previewPaths.some((p: string) => p.includes('/pages/{pageID}/preview'))).toBe(true);
		const gbPaths = body.endpoints.global_blocks.map((e: { path: string }) => e.path);
		expect(gbPaths.some((p: string) => p.includes('/api/agent/global/'))).toBe(true);

		// Block schema catalogue lists hero with text_keys.
		expect(Array.isArray(body.block_schemas)).toBe(true);
		const hero = body.block_schemas.find((s: { block_type: string }) => s.block_type === 'hero');
		expect(hero).toBeTruthy();
		const heroKeys = hero.text_keys.map((k: { key: string }) => k.key);
		expect(heroKeys).toContain('headline');
		expect(heroKeys).toContain('cta_text');

		// Editing modes documented
		expect(Array.isArray(body.editing_modes)).toBe(true);
		const modeNames = body.editing_modes.map((m: { name: string }) => m.name);
		expect(modeNames).toContain('Blocks');
		expect(modeNames).toContain('Text mode');
		expect(modeNames).toContain('Global blocks');
	});
});

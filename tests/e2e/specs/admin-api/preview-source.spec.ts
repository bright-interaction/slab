import { test, expect, u } from '../../fixtures/auth';
import {
	createBlock,
	createPage,
	createSite,
	deleteSite,
	type Site,
	type Page_
} from '../../fixtures/data';

test.describe('admin-api: block + page preview source', () => {
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

	test('GET block preview returns rendered Astro source', async ({ adminApi }) => {
		const block = await createBlock(adminApi, site.id, page.id, {
			block_type: 'hero',
			data: {
				heading: 'A calm site',
				subheading: 'that earns the click',
				cta_text: 'Get started',
				cta_url: '/start'
			}
		});
		const res = await adminApi.get(
			u(`/api/sites/${site.id}/pages/${page.id}/blocks/${block.id}/preview`)
		);
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.block_id).toBe(block.id);
		expect(body.block_type).toBe('hero');
		expect(body.page_id).toBe(page.id);
		expect(body.astro).toContain('<h2>A calm site</h2>');
		expect(body.astro).toContain('<p>that earns the click</p>');
		expect(body.astro).toContain('href="/start"');
		expect(body.astro).toContain('>Get started</a>');
	});

	test('GET block preview rejects cross-site fetch with 404', async ({ adminApi }) => {
		const block = await createBlock(adminApi, site.id, page.id);
		const res = await adminApi.get(
			u(`/api/sites/${otherSite.id}/pages/${page.id}/blocks/${block.id}/preview`)
		);
		expect(res.status()).toBe(404);
	});

	test('GET block preview returns 404 for unknown block id', async ({ adminApi }) => {
		const res = await adminApi.get(
			u(`/api/sites/${site.id}/pages/${page.id}/blocks/does-not-exist/preview`)
		);
		expect(res.status()).toBe(404);
	});

	test('GET block preview escapes HTML metacharacters in data', async ({ adminApi }) => {
		const block = await createBlock(adminApi, site.id, page.id, {
			block_type: 'text',
			data: { heading: '<script>alert(1)</script>', text: 'AT&T' }
		});
		const res = await adminApi.get(
			u(`/api/sites/${site.id}/pages/${page.id}/blocks/${block.id}/preview`)
		);
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.astro).not.toContain('<script>');
		expect(body.astro).toContain('&lt;script&gt;alert(1)&lt;/script&gt;');
		expect(body.astro).toContain('AT&amp;T');
	});

	test('GET block preview rewrites javascript: URLs to safe anchors', async ({ adminApi }) => {
		const block = await createBlock(adminApi, site.id, page.id, {
			block_type: 'cta',
			data: { cta_text: 'Click', cta_url: 'javascript:alert(1)' }
		});
		const res = await adminApi.get(
			u(`/api/sites/${site.id}/pages/${page.id}/blocks/${block.id}/preview`)
		);
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.astro).not.toContain('javascript:');
		expect(body.astro).toContain('href="#"');
	});

	test('GET page preview returns assembled .astro source', async ({ adminApi }) => {
		const fresh = await createPage(adminApi, site.id, { title: 'About', slug: 'about' });
		await createBlock(adminApi, site.id, fresh.id, {
			block_type: 'hero',
			data: { heading: 'First section' },
			sort_order: 0
		});
		await createBlock(adminApi, site.id, fresh.id, {
			block_type: 'text',
			data: { heading: 'Second section' },
			sort_order: 1
		});
		const res = await adminApi.get(u(`/api/sites/${site.id}/pages/${fresh.id}/preview`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.page_id).toBe(fresh.id);
		expect(body.slug.replace(/^\//, '')).toBe('about');
		expect(body.astro).toContain("import Base from '../layouts/Base.astro'");
		expect(body.astro).toContain('<Base title=');
		const firstIdx = body.astro.indexOf('<h2>First section</h2>');
		const secondIdx = body.astro.indexOf('<h2>Second section</h2>');
		expect(firstIdx).toBeGreaterThan(-1);
		expect(secondIdx).toBeGreaterThan(firstIdx);
	});

	test('GET page preview rejects cross-site fetch with 404', async ({ adminApi }) => {
		const res = await adminApi.get(u(`/api/sites/${otherSite.id}/pages/${page.id}/preview`));
		expect(res.status()).toBe(404);
	});

	test('GET page preview returns 404 for unknown page id', async ({ adminApi }) => {
		const res = await adminApi.get(u(`/api/sites/${site.id}/pages/missing-page/preview`));
		expect(res.status()).toBe(404);
	});

	test('Page preview reflects PATCH edits without a build', async ({ adminApi }) => {
		const fresh = await createPage(adminApi, site.id);
		const block = await createBlock(adminApi, site.id, fresh.id, {
			block_type: 'hero',
			data: { heading: 'Original' }
		});
		// First fetch
		let res = await adminApi.get(u(`/api/sites/${site.id}/pages/${fresh.id}/preview`));
		let body = await res.json();
		expect(body.astro).toContain('<h2>Original</h2>');

		// Edit via PATCH (the same path Text Mode autosave hits) and re-fetch.
		const patch = await adminApi.patch(
			u(`/api/sites/${site.id}/pages/${fresh.id}/blocks/${block.id}`),
			{ data: { data: { heading: 'Edited live' } } }
		);
		expect(patch.ok()).toBe(true);

		res = await adminApi.get(u(`/api/sites/${site.id}/pages/${fresh.id}/preview`));
		body = await res.json();
		expect(body.astro).toContain('<h2>Edited live</h2>');
		expect(body.astro).not.toContain('<h2>Original</h2>');
	});

	test('Hidden blocks do not appear in page preview', async ({ adminApi }) => {
		const fresh = await createPage(adminApi, site.id);
		await createBlock(adminApi, site.id, fresh.id, {
			block_type: 'text',
			data: { heading: 'Visible row' }
		});
		const hidden = await createBlock(adminApi, site.id, fresh.id, {
			block_type: 'text',
			data: { heading: 'Hidden row' }
		});
		await adminApi.patch(
			u(`/api/sites/${site.id}/pages/${fresh.id}/blocks/${hidden.id}`),
			{ data: { is_visible: false } }
		);
		const res = await adminApi.get(u(`/api/sites/${site.id}/pages/${fresh.id}/preview`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.astro).toContain('<h2>Visible row</h2>');
		expect(body.astro).not.toContain('<h2>Hidden row</h2>');
	});
});

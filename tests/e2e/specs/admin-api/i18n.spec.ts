import { test, expect, u } from '../../fixtures/auth';
import { createSite, createPage, deleteSite, type Site } from '../../fixtures/data';

test.describe('admin-api: i18n hreflang emission + meta-template expansion', () => {
	const cleanup: string[] = [];
	let site: Site;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('Page preview embeds hreflang alternates when locale counterparts exist', async ({
		adminApi
	}) => {
		// Declare additional_langs so the i18n config picks up sv as a valid locale.
		const upsert = await adminApi.put(u(`/api/sites/${site.id}/settings/bulk`), {
			data: [
				{ category: 'general', key: 'additional_langs', value: 'sv' },
				{ category: 'seo', key: 'hreflang_strategy', value: 'path' }
			]
		});
		expect(upsert.ok()).toBe(true);

		// Two counterparts: /about (default English) and /sv/about (Swedish).
		const enPage = await createPage(adminApi, site.id, {
			title: 'About',
			slug: 'about'
		});
		await createPage(adminApi, site.id, {
			title: 'Om oss',
			slug: 'sv/about'
		});

		// Preview the English page; it should carry the alternates prop.
		const res = await adminApi.get(u(`/api/sites/${site.id}/pages/${enPage.id}/preview`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.astro).toContain('alternates={[');
		expect(body.astro).toContain('lang: "en"');
		expect(body.astro).toContain('lang: "sv"');
		expect(body.astro).toContain('lang: "x-default"');
	});

	test('Page preview omits hreflang when no counterparts exist', async ({ adminApi }) => {
		const fresh = await createPage(adminApi, site.id, {
			title: 'Solo',
			slug: 'solo'
		});
		const res = await adminApi.get(u(`/api/sites/${site.id}/pages/${fresh.id}/preview`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		// Page exists only in default lang; no /sv/solo, so no alternates.
		expect(body.astro).not.toContain('alternates={[');
	});

	test('Meta-title template substitutes tokens before passing to Base', async ({ adminApi }) => {
		await adminApi.put(u(`/api/sites/${site.id}/settings/bulk`), {
			data: [
				{ category: 'seo', key: 'meta_title_template', value: '{page_title} | Acme Co' }
			]
		});

		const fresh = await createPage(adminApi, site.id, {
			title: 'Pricing',
			slug: 'pricing'
		});
		const res = await adminApi.get(u(`/api/sites/${site.id}/pages/${fresh.id}/preview`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.astro).toContain('title="Pricing | Acme Co"');
	});

	test('Empty meta-title template falls back to page title verbatim', async ({ adminApi }) => {
		await adminApi.put(u(`/api/sites/${site.id}/settings/bulk`), {
			data: [{ category: 'seo', key: 'meta_title_template', value: '' }]
		});
		const fresh = await createPage(adminApi, site.id, {
			title: 'Plain',
			slug: 'plain'
		});
		const res = await adminApi.get(u(`/api/sites/${site.id}/pages/${fresh.id}/preview`));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.astro).toContain('title="Plain"');
	});
});

test.describe('agent-api: i18n context surfaced to agents', () => {
	const cleanup: string[] = [];
	let site: Site;
	let agentKey: string;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);

		await adminApi.put(u(`/api/sites/${site.id}/settings/bulk`), {
			data: [
				{ category: 'general', key: 'additional_langs', value: 'sv,de' },
				{ category: 'seo', key: 'hreflang_strategy', value: 'path' },
				{ category: 'seo', key: 'meta_title_template', value: '{page_title} | {site_name}' },
				{
					category: 'seo',
					key: 'canonical_base',
					value: 'https://example.com'
				}
			]
		});
		await createPage(adminApi, site.id, { title: 'About', slug: 'about' });
		await createPage(adminApi, site.id, { title: 'Om oss', slug: 'sv/about' });

		const keyRes = await adminApi.post(u(`/api/sites/${site.id}/agent-keys`), {
			data: { name: 'i18n-context-probe', capabilities: ['read', 'write'] }
		});
		const body = await keyRes.json();
		agentKey = body.key;
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('agent context.i18n carries default_lang, additional_langs, strategy, locale_roots, templates', async ({
		adminApi
	}) => {
		const res = await adminApi.get(u('/api/agent/context'), {
			headers: { 'X-Agent-Key': agentKey }
		});
		expect(res.ok()).toBe(true);
		const body = await res.json();
		const i = body.i18n;
		expect(i).toBeTruthy();
		expect(i.default_lang).toBeTruthy();
		expect(i.additional_langs).toEqual(expect.arrayContaining(['sv', 'de']));
		expect(i.hreflang_strategy).toBe('path');
		expect(i.locale_roots).toEqual(expect.arrayContaining(['', '/sv']));
		expect(i.meta_title_template).toBe('{page_title} | {site_name}');
		expect(i.canonical_base).toBe('https://example.com');
	});
});

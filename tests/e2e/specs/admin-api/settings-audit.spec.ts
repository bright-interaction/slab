import { test, expect, u } from '../../fixtures/auth';
import { createSite, deleteSite, type Site } from '../../fixtures/data';

// Settings audit. Locks in the wiring fixes shipped 2026-04-29 so dead
// settings can't reappear silently:
//
//   - SettingsCatalog block in /api/agent/context lists every writable key
//     with type + valid range + writable flag + admin URL.
//   - Validation rejects bad enums / formats / ranges with 422.
//   - The previously-dead settings (domain_aliases, og_default_image_id,
//     ga4_id, umami_*, robots_txt_extra, sitemap_enabled) are now reachable
//     via the same PATCH path the agent uses.

test.describe('admin-api: settings_catalog + validation', () => {
	const cleanup: string[] = [];
	let site: Site;
	let agentKey: string;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);

		const keyRes = await adminApi.post(u(`/api/sites/${site.id}/agent-keys`), {
			data: { name: 'settings-audit-probe', capabilities: ['read', 'write'] }
		});
		const body = await keyRes.json();
		agentKey = body.key;
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('agent context.settings_catalog enumerates every writable + admin-only setting', async ({
		adminApi
	}) => {
		const res = await adminApi.get(u('/api/agent/context'), {
			headers: { 'X-Agent-Key': agentKey }
		});
		expect(res.ok()).toBe(true);
		const body = await res.json();
		const cat = body.settings_catalog;
		expect(cat).toBeTruthy();
		expect(cat.writable_categories).toEqual(
			expect.arrayContaining(['analytics', 'general', 'seo'])
		);
		expect(cat.admin_only_categories).toEqual(expect.arrayContaining(['security']));
		const keys = (cat.items as Array<{ category: string; key: string }>).map(
			(i) => `${i.category}.${i.key}`
		);
		// Spot-check the previously-dead settings.
		for (const fqn of [
			'general.domain_aliases',
			'seo.og_default_image_id',
			'seo.sitemap_enabled',
			'seo.robots_txt_extra',
			'analytics.ga4_id',
			'analytics.umami_url',
			'analytics.umami_site_id',
			'analytics.cookie_banner_snippet',
			'security.x_frame_options',
			'security.referrer_policy'
		]) {
			expect(keys).toContain(fqn);
		}
		// Each item carries the metadata an agent needs to write safely.
		const sample = cat.items.find(
			(i: { category: string; key: string }) =>
				i.category === 'seo' && i.key === 'hreflang_strategy'
		);
		expect(sample).toBeTruthy();
		expect(sample.value_type).toBe('enum');
		expect(sample.enum_values).toEqual(['path', 'subdomain', 'off']);
		expect(sample.agent_writable).toBe(true);
		expect(sample.human_admin_url).toContain('/sites/');
		expect(sample.human_admin_url).toContain('/settings/seo');
	});

	test('admin BulkUpsert rejects invalid enum with 422', async ({ adminApi }) => {
		const res = await adminApi.put(u(`/api/sites/${site.id}/settings/bulk`), {
			data: [{ category: 'security', key: 'x_frame_options', value: 'GARBAGE' }]
		});
		expect(res.status()).toBe(422);
		const err = await res.json();
		expect(err.error || err.message).toMatch(/x_frame_options/);
	});

	test('agent BulkUpsert rejects invalid range with 422', async ({ adminApi }) => {
		const res = await adminApi.patch(u('/api/agent/settings'), {
			headers: { 'X-Agent-Key': agentKey },
			data: [{ category: 'analytics', key: 'identity_max_age_days', value: '99999' }]
		});
		expect(res.status()).toBe(422);
	});

	test('agent BulkUpsert rejects malformed URL with 422', async ({ adminApi }) => {
		const res = await adminApi.patch(u('/api/agent/settings'), {
			headers: { 'X-Agent-Key': agentKey },
			data: [{ category: 'analytics', key: 'umami_url', value: 'not a url' }]
		});
		expect(res.status()).toBe(422);
	});

	test('agent BulkUpsert rejects bad GA4 ID with 422', async ({ adminApi }) => {
		const res = await adminApi.patch(u('/api/agent/settings'), {
			headers: { 'X-Agent-Key': agentKey },
			data: [{ category: 'analytics', key: 'ga4_id', value: 'XX-12345' }]
		});
		expect(res.status()).toBe(422);
	});

	test('valid bulk write succeeds end-to-end and round-trips', async ({ adminApi }) => {
		const res = await adminApi.patch(u('/api/agent/settings'), {
			headers: { 'X-Agent-Key': agentKey },
			data: [
				{ category: 'analytics', key: 'ga4_enabled', value: '1' },
				{ category: 'analytics', key: 'ga4_id', value: 'G-ABC1234' },
				{ category: 'analytics', key: 'umami_enabled', value: '1' },
				{ category: 'analytics', key: 'umami_url', value: 'https://analytics.example.com' },
				{ category: 'analytics', key: 'umami_site_id', value: 'abc-123' },
				{ category: 'analytics', key: 'identity_max_age_days', value: '14' },
				{ category: 'general', key: 'domain_aliases', value: 'www.example.com,example.org' },
				{ category: 'seo', key: 'hreflang_strategy', value: 'subdomain' },
				{ category: 'seo', key: 'sitemap_enabled', value: '0' },
				{ category: 'seo', key: 'robots_txt_extra', value: 'Disallow: /private/' }
			]
		});
		expect(res.ok()).toBe(true);
		const ctx = await adminApi.get(u('/api/agent/context'), {
			headers: { 'X-Agent-Key': agentKey }
		});
		const body = await ctx.json();
		const items = body.settings_catalog.items as Array<{
			category: string;
			key: string;
			current_value: string;
			is_default: boolean;
		}>;
		const ga4 = items.find((i) => i.category === 'analytics' && i.key === 'ga4_id');
		expect(ga4?.current_value).toBe('G-ABC1234');
		expect(ga4?.is_default).toBe(false);
		const aliases = items.find((i) => i.category === 'general' && i.key === 'domain_aliases');
		expect(aliases?.current_value).toBe('www.example.com,example.org');
	});

	test('admin-only setting rejected at agent path (rejected_admin_only)', async ({ adminApi }) => {
		const res = await adminApi.patch(u('/api/agent/settings'), {
			headers: { 'X-Agent-Key': agentKey },
			data: [{ category: 'security', key: 'hsts_enabled', value: '1' }]
		});
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.rejected_admin_only).toContain('security.hsts_enabled');
	});
});

import { test, expect, u } from '../../fixtures/auth';
import {
	createSite,
	createPage,
	deleteSite,
	createAgentKey,
	agentApi,
	rand,
	type Site,
	type Page_
} from '../../fixtures/data';

test.describe('guardrails: allow_block_type whitelist', () => {
	const cleanup: string[] = [];
	let site: Site;
	let page: Page_;
	let agentKey: string;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
		page = await createPage(adminApi, site.id, { slug: rand('abt') });
		const key = await createAgentKey(adminApi, site.id, ['read', 'write', 'build']);
		agentKey = key.key;

		// Whitelist hero + text only.
		await adminApi.post(u(`/api/sites/${site.id}/guardrails`), {
			data: { rule_type: 'allow_block_type', target: '*', value: 'hero', severity: 'error' }
		});
		await adminApi.post(u(`/api/sites/${site.id}/guardrails`), {
			data: { rule_type: 'allow_block_type', target: '*', value: 'text', severity: 'error' }
		});
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('block_type=cta is rejected when not whitelisted', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u(`/api/agent/pages/${page.slug.replace(/^\//, '')}/blocks`), {
				data: { block_type: 'cta', data: { label: 'Click' }, sort_order: 0 }
			});
			expect(res.status()).toBe(422);
			const body = await res.json();
			expect(
				body.violations.some(
					(v: { rule: string }) => v.rule === 'allow_block_type'
				)
			).toBe(true);
		} finally {
			await ctx.dispose();
		}
	});

	test('block_type=text is allowed (whitelisted)', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u(`/api/agent/pages/${page.slug.replace(/^\//, '')}/blocks`), {
				data: { block_type: 'text', data: { content: 'allowed' }, sort_order: 1 }
			});
			expect(res.status()).toBe(201);
		} finally {
			await ctx.dispose();
		}
	});

	test('block_type=hero is allowed (whitelisted)', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u(`/api/agent/pages/${page.slug.replace(/^\//, '')}/blocks`), {
				data: { block_type: 'hero', data: { headline: 'Hi' }, sort_order: 2 }
			});
			expect(res.status()).toBe(201);
		} finally {
			await ctx.dispose();
		}
	});
});

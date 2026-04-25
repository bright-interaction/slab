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

test.describe('guardrails: forbid_pattern', () => {
	const cleanup: string[] = [];
	let site: Site;
	let page: Page_;
	let agentKey: string;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
		page = await createPage(adminApi, site.id, { slug: rand('fp') });
		const key = await createAgentKey(adminApi, site.id, ['read', 'write', 'build']);
		agentKey = key.key;

		await adminApi.post(u(`/api/sites/${site.id}/guardrails`), {
			data: {
				rule_type: 'forbid_pattern',
				target: '*',
				value: 'SHOULDNOTAPPEAR',
				severity: 'error'
			}
		});
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('block whose data contains the forbidden string is rejected', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u(`/api/agent/pages/${page.slug.replace(/^\//, '')}/blocks`), {
				data: {
					block_type: 'text',
					data: { content: 'this has SHOULDNOTAPPEAR inside it' },
					sort_order: 0
				}
			});
			expect(res.status()).toBe(422);
			const body = await res.json();
			expect(
				body.violations.some(
					(v: { rule: string }) => v.rule === 'forbid_pattern'
				)
			).toBe(true);
		} finally {
			await ctx.dispose();
		}
	});

	test('block without the forbidden string succeeds', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u(`/api/agent/pages/${page.slug.replace(/^\//, '')}/blocks`), {
				data: {
					block_type: 'text',
					data: { content: 'this is fine' },
					sort_order: 1
				}
			});
			expect(res.status()).toBe(201);
		} finally {
			await ctx.dispose();
		}
	});
});

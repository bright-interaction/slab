import { test, expect, u } from '../../fixtures/auth';
import {
	createSite,
	deleteSite,
	createAgentKey,
	agentApi,
	type Site
} from '../../fixtures/data';

test.describe('guardrails: max_url_depth', () => {
	const cleanup: string[] = [];
	let site: Site;
	let agentKey: string;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
		const key = await createAgentKey(adminApi, site.id, ['read', 'write', 'build']);
		agentKey = key.key;
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('page with 4 path segments is rejected (max_depth=3)', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u('/api/agent/pages'), {
				data: { title: 'Too Deep', slug: '/a/b/c/d', layout: 'default' }
			});
			expect(res.status()).toBe(422);
			const body = await res.json();
			expect(
				body.violations.some(
					(v: { rule: string }) => v.rule === 'max_url_depth'
				)
			).toBe(true);
		} finally {
			await ctx.dispose();
		}
	});

	test('page with 3 segments is allowed', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u('/api/agent/pages'), {
				data: { title: 'Three Deep', slug: '/x/y/z', layout: 'default' }
			});
			expect(res.status()).toBe(201);
		} finally {
			await ctx.dispose();
		}
	});

	test('uppercase slug is rejected by url_convention rule', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u('/api/agent/pages'), {
				data: { title: 'Upper', slug: '/Foo', layout: 'default' }
			});
			expect(res.status()).toBe(422);
			const body = await res.json();
			expect(
				body.violations.some(
					(v: { rule: string }) => v.rule === 'url_convention'
				)
			).toBe(true);
		} finally {
			await ctx.dispose();
		}
	});

	test('slug with underscore is rejected', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u('/api/agent/pages'), {
				data: { title: 'Underscored', slug: '/foo_bar', layout: 'default' }
			});
			expect(res.status()).toBe(422);
		} finally {
			await ctx.dispose();
		}
	});
});

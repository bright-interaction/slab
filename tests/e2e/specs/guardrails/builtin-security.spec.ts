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

test.describe('guardrails: built-in security checks', () => {
	const cleanup: string[] = [];
	let site: Site;
	let page: Page_;
	let agentKey: string;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
		page = await createPage(adminApi, site.id, { slug: rand('sec') });
		const key = await createAgentKey(adminApi, site.id, ['read', 'write', 'build']);
		agentKey = key.key;
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('agent rejects block with <script> tag', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u(`/api/agent/pages/${page.slug.replace(/^\//, '')}/blocks`), {
				data: {
					block_type: 'text',
					data: { content: 'hi <script>alert(1)</script>' },
					sort_order: 0
				}
			});
			expect(res.status()).toBe(422);
			const body = await res.json();
			expect(body.error).toMatch(/Guardrail violations/i);
			expect(Array.isArray(body.violations)).toBe(true);
			expect(
				body.violations.some(
					(v: { rule: string }) => v.rule === 'security'
				)
			).toBe(true);
		} finally {
			await ctx.dispose();
		}
	});

	test('agent rejects block with javascript: URL', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u(`/api/agent/pages/${page.slug.replace(/^\//, '')}/blocks`), {
				data: {
					block_type: 'text',
					data: { href: 'javascript:alert(1)' },
					sort_order: 1
				}
			});
			expect(res.status()).toBe(422);
			const body = await res.json();
			expect(body.violations.some((v: { rule: string }) => v.rule === 'security')).toBe(true);
		} finally {
			await ctx.dispose();
		}
	});

	test('agent rejects block with onclick= handler', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u(`/api/agent/pages/${page.slug.replace(/^\//, '')}/blocks`), {
				data: {
					block_type: 'text',
					data: { html: '<div onclick=alert(1)>hi</div>' },
					sort_order: 2
				}
			});
			expect(res.status()).toBe(422);
		} finally {
			await ctx.dispose();
		}
	});

	test('agent rejects block with onerror= handler', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u(`/api/agent/pages/${page.slug.replace(/^\//, '')}/blocks`), {
				data: {
					block_type: 'image',
					data: { html: '<img src=x onerror=alert(1)>' },
					sort_order: 3
				}
			});
			expect(res.status()).toBe(422);
		} finally {
			await ctx.dispose();
		}
	});

	test('agent allows clean block with only safe markup', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u(`/api/agent/pages/${page.slug.replace(/^\//, '')}/blocks`), {
				data: {
					block_type: 'text',
					data: { content: 'Hello world, no scripts here.' },
					sort_order: 4
				}
			});
			expect(res.status()).toBe(201);
		} finally {
			await ctx.dispose();
		}
	});

	test('admin endpoint for block creation does NOT enforce guardrails (known)', async ({
		adminApi
	}) => {
		// Admin block-create handler bypasses GuardrailEngine; <script> goes
		// through. This documents the actual behavior so we notice if it changes.
		const res = await adminApi.post(u(`/api/sites/${site.id}/pages/${page.id}/blocks`), {
			data: {
				block_type: 'text',
				data: { content: '<script>admin</script>' },
				sort_order: 99
			}
		});
		expect(res.status()).toBe(201);
	});
});

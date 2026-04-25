import { test, expect, u } from '../../fixtures/auth';
import {
	createSite,
	deleteSite,
	createAgentKey,
	agentApi,
	rand,
	type Site
} from '../../fixtures/data';

test.describe('agent-api: capability scoping', () => {
	const cleanup: string[] = [];
	let site: Site;
	let readOnlyKey: string;
	let fullKey: string;

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
		const ro = await createAgentKey(adminApi, site.id, ['read']);
		readOnlyKey = ro.key;
		const full = await createAgentKey(adminApi, site.id, ['read', 'write', 'build']);
		fullKey = full.key;
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('read-only key can GET /api/agent/context (read succeeds)', async () => {
		const ctx = await agentApi(readOnlyKey);
		try {
			const res = await ctx.get(u('/api/agent/context'));
			expect(res.ok()).toBe(true);
		} finally {
			await ctx.dispose();
		}
	});

	test('read-only key cannot POST /api/agent/pages (write rejected)', async () => {
		const ctx = await agentApi(readOnlyKey);
		try {
			const res = await ctx.post(u('/api/agent/pages'), {
				data: { title: 'Should Fail', slug: `/${rand('ro')}`, layout: 'default' }
			});
			// Backend uses 403 for capability mismatch.
			expect([401, 403]).toContain(res.status());
		} finally {
			await ctx.dispose();
		}
	});

	test('read-only key cannot POST /api/agent/blocks', async () => {
		// First create a page using the full key.
		const fullCtx = await agentApi(fullKey);
		const slug = rand('ro2');
		try {
			await fullCtx.post(u('/api/agent/pages'), {
				data: { title: 'Host', slug: `/${slug}`, layout: 'default' }
			});
		} finally {
			await fullCtx.dispose();
		}

		const ctx = await agentApi(readOnlyKey);
		try {
			const res = await ctx.post(u(`/api/agent/pages/${slug}/blocks`), {
				data: { block_type: 'text', data: { content: 'nope' }, sort_order: 0 }
			});
			expect([401, 403]).toContain(res.status());
		} finally {
			await ctx.dispose();
		}
	});

	test('read-only key cannot POST /api/agent/components', async () => {
		const ctx = await agentApi(readOnlyKey);
		try {
			const res = await ctx.post(u('/api/agent/components'), {
				data: {
					name: rand('roc'),
					category: 'section',
					template: '<div/>',
					props_schema: {},
					css_classes: []
				}
			});
			expect([401, 403]).toContain(res.status());
		} finally {
			await ctx.dispose();
		}
	});

	test('read-only key cannot PUT /api/agent/global/{slot}', async () => {
		const ctx = await agentApi(readOnlyKey);
		try {
			const res = await ctx.put(u('/api/agent/global/header'), {
				data: { block_type: 'header', data: { links: [] } }
			});
			expect([401, 403]).toContain(res.status());
		} finally {
			await ctx.dispose();
		}
	});

	test('full key (read+write) can POST /api/agent/pages', async () => {
		const ctx = await agentApi(fullKey);
		try {
			const res = await ctx.post(u('/api/agent/pages'), {
				data: { title: 'OK', slug: `/${rand('full')}`, layout: 'default' }
			});
			expect(res.status()).toBe(201);
		} finally {
			await ctx.dispose();
		}
	});

	test('revoked key returns 401 on subsequent calls', async ({ adminApi }) => {
		const tempKey = await createAgentKey(adminApi, site.id, ['read', 'write']);
		const ctx = await agentApi(tempKey.key);
		try {
			// Sanity: works pre-revocation.
			const ok = await ctx.get(u('/api/agent/context'));
			expect(ok.ok()).toBe(true);

			const revoke = await adminApi.delete(u(`/api/sites/${site.id}/agent-keys/${tempKey.id}`));
			expect(revoke.ok()).toBe(true);

			const denied = await ctx.get(u('/api/agent/context'));
			expect(denied.status()).toBe(401);
		} finally {
			await ctx.dispose();
		}
	});
});

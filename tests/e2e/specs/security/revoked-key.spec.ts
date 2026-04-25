import { test, expect, u } from '../../fixtures/auth';
import {
	agentApi,
	createAgentKey,
	createSite,
	deleteSite,
	type Site
} from '../../fixtures/data';

// DeactivateAgentKey flips is_active=0. GetAgentKeyByHash filters on
// is_active=1, so a revoked key looks like an unknown hash to the middleware
// and answers 401.

test.describe('security: revoked agent key returns 401', () => {
	let site: Site;
	const cleanup: string[] = [];

	test.beforeAll(async ({ adminApi }) => {
		site = await createSite(adminApi);
		cleanup.push(site.id);
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('using a revoked key returns 401', async ({ adminApi }) => {
		const generated = await createAgentKey(adminApi, site.id, ['read']);
		// Sanity: the key works pre-revoke.
		{
			const ctx = await agentApi(generated.key);
			try {
				const ok = await ctx.get(u('/api/agent/context'));
				expect(ok.ok(), `expected pre-revoke key to work, got ${ok.status()}`).toBe(true);
			} finally {
				await ctx.dispose();
			}
		}

		// Revoke via admin endpoint.
		const del = await adminApi.delete(u(`/api/sites/${site.id}/agent-keys/${generated.id}`));
		expect(del.ok()).toBe(true);

		// Now the same raw key must fail.
		const ctx = await agentApi(generated.key);
		try {
			const res = await ctx.get(u('/api/agent/context'));
			expect(res.status()).toBe(401);
		} finally {
			await ctx.dispose();
		}
	});
});

import { test, expect, u } from '../../fixtures/auth';
import {
	agentApi,
	createAgentKey,
	createPage,
	createSite,
	deleteSite,
	type Site
} from '../../fixtures/data';

// Agent endpoints are scoped to the agent key's SiteID. There is no URL
// parameter to specify a different site. We verify:
//   - Agent A's key sees only Site A's pages, never Site B's, in context().
//   - Block IDs from Site B are not addressable via Agent A's key.

test.describe('security: agent key cannot read another site\'s data', () => {
	const cleanup: string[] = [];
	let siteA: Site;
	let siteB: Site;
	let keyA: string;

	test.beforeAll(async ({ adminApi }) => {
		siteA = await createSite(adminApi);
		siteB = await createSite(adminApi);
		cleanup.push(siteA.id, siteB.id);

		// Distinguishable page on each site.
		await createPage(adminApi, siteA.id, { slug: 'a-only', title: 'A only' });
		await createPage(adminApi, siteB.id, { slug: 'b-only', title: 'B only' });

		const k = await createAgentKey(adminApi, siteA.id, ['read', 'write', 'build']);
		keyA = k.key;
	});

	test.afterAll(async ({ adminApi }) => {
		for (const id of cleanup) await deleteSite(adminApi, id);
	});

	test('GET /api/agent/context with key A returns only site A', async () => {
		const ctx = await agentApi(keyA);
		try {
			const res = await ctx.get(u('/api/agent/context'));
			expect(res.ok()).toBe(true);
			const body = await res.json();
			expect(body.site.id).toBe(siteA.id);
			expect(body.site.id).not.toBe(siteB.id);

			const slugs = (body.structure.pages as Array<{ slug: string }>).map((p) => p.slug);
			expect(slugs).toContain('/a-only');
			expect(slugs).not.toContain('/b-only');
		} finally {
			await ctx.dispose();
		}
	});

	test('PATCH on a page slug that exists only on site B returns 404 for agent A', async () => {
		const ctx = await agentApi(keyA);
		try {
			const res = await ctx.patch(u('/api/agent/pages/b-only'), {
				data: { title: 'pwned' }
			});
			expect([404, 403]).toContain(res.status());
		} finally {
			await ctx.dispose();
		}
	});

	test('DELETE site B via admin endpoint with agent A key fails (agent has no admin scope)', async () => {
		const ctx = await agentApi(keyA);
		try {
			const res = await ctx.delete(u(`/api/sites/${siteB.id}`));
			// Agent key auth doesn't satisfy admin-cookie middleware -> 401.
			expect(res.status()).toBe(401);
		} finally {
			await ctx.dispose();
		}
	});
});

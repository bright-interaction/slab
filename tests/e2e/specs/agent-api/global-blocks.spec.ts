import { test, expect, u } from '../../fixtures/auth';
import {
	createSite,
	deleteSite,
	createAgentKey,
	agentApi,
	type Site
} from '../../fixtures/data';

test.describe('agent-api: PUT /api/agent/global/{slot}', () => {
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

	test('updates the seeded header global block', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.put(u('/api/agent/global/header'), {
				data: {
					name: 'Custom Header',
					block_type: 'header',
					data: {
						links: [
							{ label: 'Home', href: '/' },
							{ label: 'Pricing', href: '/pricing' }
						]
					}
				}
			});
			expect(res.ok()).toBe(true);
			const gb = await res.json();
			expect(gb.slot).toBe('header');
			expect(gb.name).toBe('Custom Header');
		} finally {
			await ctx.dispose();
		}
	});

	test('updates the seeded footer global block', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.put(u('/api/agent/global/footer'), {
				data: {
					block_type: 'footer',
					data: {
						copyright: 'Updated 2026',
						links: [{ label: 'Privacy', href: '/privacy' }]
					}
				}
			});
			expect(res.ok()).toBe(true);
			const gb = await res.json();
			expect(gb.slot).toBe('footer');
		} finally {
			await ctx.dispose();
		}
	});

	test('creates a new global block when slot does not exist', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.put(u('/api/agent/global/nav'), {
				data: {
					name: 'Side Nav',
					block_type: 'nav',
					data: {
						items: [{ label: 'Docs', href: '/docs' }]
					}
				}
			});
			// Either 200 (updated) or 201 (created). Both fine.
			expect([200, 201]).toContain(res.status());
			const gb = await res.json();
			expect(gb.slot).toBe('nav');
		} finally {
			await ctx.dispose();
		}
	});

	test('verifies header survives via /api/agent/context', async () => {
		const ctx = await agentApi(agentKey);
		try {
			await ctx.put(u('/api/agent/global/header'), {
				data: {
					name: 'CtxCheck Header',
					block_type: 'header',
					data: { links: [{ label: 'Home', href: '/' }] }
				}
			});
			const res = await ctx.get(u('/api/agent/context'));
			const body = await res.json();
			const header = body.structure.global_blocks.find(
				(gb: { slot: string }) => gb.slot === 'header'
			);
			expect(header).toBeTruthy();
			expect(header.name).toBe('CtxCheck Header');
		} finally {
			await ctx.dispose();
		}
	});
});

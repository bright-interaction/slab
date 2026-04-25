import { test, expect, u } from '../../fixtures/auth';
import {
	createSite,
	deleteSite,
	createAgentKey,
	agentApi,
	rand,
	type Site
} from '../../fixtures/data';

test.describe('agent-api: pages and blocks', () => {
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

	test('POST /api/agent/pages creates a page', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const slug = `/${rand('p')}`;
			const res = await ctx.post(u('/api/agent/pages'), {
				data: { title: 'Agent Created', slug, layout: 'default' }
			});
			expect(res.status()).toBe(201);
			const page = await res.json();
			expect(page.id).toBeTruthy();
			expect(page.slug).toBe(slug);
			expect(page.title).toBe('Agent Created');
		} finally {
			await ctx.dispose();
		}
	});

	test('PATCH /api/agent/pages/{slug} updates page via slug', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const slug = rand('p2');
			const create = await ctx.post(u('/api/agent/pages'), {
				data: { title: 'Old Title', slug: `/${slug}`, layout: 'default' }
			});
			expect(create.status()).toBe(201);

			const res = await ctx.patch(u(`/api/agent/pages/${slug}`), {
				data: { title: 'New Title' }
			});
			expect(res.ok()).toBe(true);
			const body = await res.json();
			expect(body.title).toBe('New Title');
		} finally {
			await ctx.dispose();
		}
	});

	test('DELETE /api/agent/pages/{slug} removes page', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const slug = rand('p3');
			await ctx.post(u('/api/agent/pages'), {
				data: { title: 'Doomed', slug: `/${slug}`, layout: 'default' }
			});
			const res = await ctx.delete(u(`/api/agent/pages/${slug}`));
			expect(res.ok()).toBe(true);
			const body = await res.json();
			expect(body.status).toBe('deleted');
		} finally {
			await ctx.dispose();
		}
	});

	test('POST /api/agent/blocks creates block on page via ?page= query', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const slug = rand('p4');
			await ctx.post(u('/api/agent/pages'), {
				data: { title: 'Block Host', slug: `/${slug}`, layout: 'default' }
			});
			const res = await ctx.post(u(`/api/agent/blocks?page=/${slug}`), {
				data: {
					block_type: 'text',
					data: { content: 'hello' },
					sort_order: 0
				}
			});
			expect(res.status()).toBe(201);
			const block = await res.json();
			expect(block.id).toBeTruthy();
			expect(block.block_type).toBe('text');
		} finally {
			await ctx.dispose();
		}
	});

	test('POST /api/agent/pages/{slug}/blocks also creates block', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const slug = rand('p5');
			await ctx.post(u('/api/agent/pages'), {
				data: { title: 'Path Host', slug: `/${slug}`, layout: 'default' }
			});
			const res = await ctx.post(u(`/api/agent/pages/${slug}/blocks`), {
				data: {
					block_type: 'hero',
					data: { headline: 'Welcome' },
					sort_order: 0
				}
			});
			expect(res.status()).toBe(201);
			const block = await res.json();
			expect(block.block_type).toBe('hero');
		} finally {
			await ctx.dispose();
		}
	});

	test('PATCH /api/agent/blocks/{blockID} updates a block', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const slug = rand('p6');
			await ctx.post(u('/api/agent/pages'), {
				data: { title: 'Update Host', slug: `/${slug}`, layout: 'default' }
			});
			const create = await ctx.post(u(`/api/agent/pages/${slug}/blocks`), {
				data: { block_type: 'text', data: { content: 'before' }, sort_order: 0 }
			});
			const block = await create.json();
			const res = await ctx.patch(u(`/api/agent/blocks/${block.id}`), {
				data: { data: { content: 'after' } }
			});
			expect(res.ok()).toBe(true);
			const updated = await res.json();
			expect(updated.id).toBe(block.id);
		} finally {
			await ctx.dispose();
		}
	});

	test('DELETE /api/agent/blocks/{blockID} removes a block', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const slug = rand('p7');
			await ctx.post(u('/api/agent/pages'), {
				data: { title: 'Del Host', slug: `/${slug}`, layout: 'default' }
			});
			const create = await ctx.post(u(`/api/agent/pages/${slug}/blocks`), {
				data: { block_type: 'text', data: { content: 'temp' }, sort_order: 0 }
			});
			const block = await create.json();
			const res = await ctx.delete(u(`/api/agent/blocks/${block.id}`));
			expect(res.ok()).toBe(true);
			const body = await res.json();
			expect(body.status).toBe('deleted');
		} finally {
			await ctx.dispose();
		}
	});

	test('PATCH on missing slug returns 404', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.patch(u(`/api/agent/pages/${rand('missing')}`), {
				data: { title: 'Nope' }
			});
			expect(res.status()).toBe(404);
		} finally {
			await ctx.dispose();
		}
	});
});

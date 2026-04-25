import { test, expect, u } from '../../fixtures/auth';
import {
	createSite,
	deleteSite,
	createAgentKey,
	agentApi,
	rand,
	type Site
} from '../../fixtures/data';

test.describe('agent-api: components', () => {
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

	test('POST /api/agent/components creates a component', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const name = rand('comp');
			const res = await ctx.post(u('/api/agent/components'), {
				data: {
					name,
					category: 'section',
					template: '<div class="hero">{{title}}</div>',
					props_schema: {
						type: 'object',
						properties: { title: { type: 'string' } }
					},
					css_classes: ['hero'],
					usage_note: 'Hero block'
				}
			});
			expect(res.status()).toBe(201);
			const comp = await res.json();
			expect(comp.id).toBeTruthy();
			expect(comp.name).toBe(name);
			expect(comp.category).toBe('section');
		} finally {
			await ctx.dispose();
		}
	});

	test('PATCH /api/agent/components/{name} updates fields', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const name = rand('comp2');
			const create = await ctx.post(u('/api/agent/components'), {
				data: {
					name,
					category: 'section',
					template: '<div>v1</div>',
					props_schema: { type: 'object' },
					css_classes: []
				}
			});
			expect(create.status()).toBe(201);

			const res = await ctx.patch(u(`/api/agent/components/${name}`), {
				data: {
					template: '<div>v2</div>',
					usage_note: 'Updated note'
				}
			});
			expect(res.ok()).toBe(true);
			const comp = await res.json();
			expect(comp.usage_note).toBe('Updated note');
		} finally {
			await ctx.dispose();
		}
	});

	test('props_schema persists as parsed JSON in context', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const name = rand('comp3');
			const schema = {
				type: 'object',
				properties: { headline: { type: 'string', required: true } }
			};
			const res = await ctx.post(u('/api/agent/components'), {
				data: {
					name,
					category: 'section',
					template: '<h1>{{headline}}</h1>',
					props_schema: schema,
					css_classes: ['display-1']
				}
			});
			expect(res.status()).toBe(201);

			// Verify via /api/agent/context - components show props_schema parsed.
			const ctxRes = await ctx.get(u('/api/agent/context'));
			const body = await ctxRes.json();
			const comp = body.components.find((c: { name: string }) => c.name === name);
			expect(comp).toBeTruthy();
			expect(comp.props_schema).toEqual(schema);
		} finally {
			await ctx.dispose();
		}
	});

	test('PATCH on missing component returns 404', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.patch(u(`/api/agent/components/${rand('missing')}`), {
				data: { template: '<div/>' }
			});
			expect(res.status()).toBe(404);
		} finally {
			await ctx.dispose();
		}
	});

	test('POST without name or template returns 400', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u('/api/agent/components'), {
				data: { category: 'section' }
			});
			expect(res.status()).toBe(400);
		} finally {
			await ctx.dispose();
		}
	});
});

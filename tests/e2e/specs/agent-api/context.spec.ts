import { test, expect, u } from '../../fixtures/auth';
import {
	createSite,
	deleteSite,
	createAgentKey,
	agentApi,
	type Site
} from '../../fixtures/data';

test.describe('agent-api: GET /api/agent/context', () => {
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

	test('returns full site context with all top-level keys', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.get(u('/api/agent/context'));
			expect(res.ok()).toBe(true);
			expect(res.status()).toBe(200);
			const body = await res.json();
			expect(body).toHaveProperty('site');
			expect(body).toHaveProperty('structure');
			expect(body).toHaveProperty('knowledgebase');
			expect(body).toHaveProperty('components');
			expect(body).toHaveProperty('css_classes');
			expect(body).toHaveProperty('constraints');
			expect(body).toHaveProperty('architecture');
		} finally {
			await ctx.dispose();
		}
	});

	test('site object has id, name, branding, and seo', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.get(u('/api/agent/context'));
			const body = await res.json();
			expect(body.site.id).toBe(site.id);
			expect(body.site.name).toBeTruthy();
			expect(body.site.branding).toBeTruthy();
			expect(body.site.branding.primary_color).toMatch(/^#[0-9a-fA-F]{6}$/);
			expect(body.site.seo).toBeDefined();
		} finally {
			await ctx.dispose();
		}
	});

	test('structure includes pages array, global_blocks, silos', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.get(u('/api/agent/context'));
			const body = await res.json();
			expect(body.structure).toBeTruthy();
			expect(Array.isArray(body.structure.pages)).toBe(true);
			expect(Array.isArray(body.structure.global_blocks)).toBe(true);
			// silos may be absent on one-pager sites; accept missing OR array.
			if (body.structure.silos !== undefined && body.structure.silos !== null) {
				expect(Array.isArray(body.structure.silos)).toBe(true);
			}
			// seeded sites get essential pages (privacy, terms, cookies, 404)
			expect(body.structure.pages.length).toBeGreaterThan(0);
		} finally {
			await ctx.dispose();
		}
	});

	test('constraints include max_blocks_per_page and max_url_depth', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.get(u('/api/agent/context'));
			const body = await res.json();
			expect(body.constraints).toBeTruthy();
			expect(typeof body.constraints.max_blocks_per_page).toBe('number');
			expect(typeof body.constraints.max_url_depth).toBe('number');
			expect(body.constraints.max_url_depth).toBeGreaterThan(0);
		} finally {
			await ctx.dispose();
		}
	});

	test('architecture has structure_type and max_depth', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.get(u('/api/agent/context'));
			const body = await res.json();
			expect(body.architecture).toBeTruthy();
			expect(typeof body.architecture.structure_type).toBe('string');
			expect(typeof body.architecture.max_depth).toBe('number');
		} finally {
			await ctx.dispose();
		}
	});

	test('without X-Agent-Key returns 401', async ({ adminApi }) => {
		// adminApi has cookie auth, no agent key -> agent route rejects
		const res = await adminApi.get(u('/api/agent/context'));
		expect(res.status()).toBe(401);
	});

	test('with invalid X-Agent-Key returns 401', async () => {
		const ctx = await agentApi('ask_invalid_key_value');
		try {
			const res = await ctx.get(u('/api/agent/context'));
			expect(res.status()).toBe(401);
		} finally {
			await ctx.dispose();
		}
	});
});

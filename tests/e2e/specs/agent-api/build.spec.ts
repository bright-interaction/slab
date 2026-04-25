import { test, expect, u } from '../../fixtures/auth';
import {
	createSite,
	deleteSite,
	createAgentKey,
	agentApi,
	type Site
} from '../../fixtures/data';

test.describe('agent-api: build trigger and status', () => {
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

	test('POST /api/agent/build returns build_id and status=building', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u('/api/agent/build'), { data: {} });
			expect(res.status()).toBe(202);
			const body = await res.json();
			expect(body.build_id).toBeTruthy();
			expect(body.status).toBe('building');
		} finally {
			await ctx.dispose();
		}
	});

	test('GET /api/agent/build/{buildID}/status polls to terminal state', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const triggered = await ctx.post(u('/api/agent/build'), { data: {} });
			expect(triggered.status()).toBe(202);
			const { build_id } = await triggered.json();
			expect(build_id).toBeTruthy();

			const start = Date.now();
			const timeoutMs = 120_000;
			let last: { status: string; build_log?: string; error?: string } = { status: '' };
			while (Date.now() - start < timeoutMs) {
				const res = await ctx.get(u(`/api/agent/build/${build_id}/status`));
				expect(res.ok()).toBe(true);
				last = await res.json();
				if (last.status === 'success' || last.status === 'failed' || last.status === 'timeout') {
					break;
				}
				await new Promise((r) => setTimeout(r, 750));
			}
			expect(['success', 'failed', 'timeout']).toContain(last.status);
		} finally {
			await ctx.dispose();
		}
	});

	test('GET status of unknown buildID returns 404', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.get(u('/api/agent/build/000000000000000000000000/status'));
			expect(res.status()).toBe(404);
		} finally {
			await ctx.dispose();
		}
	});
});

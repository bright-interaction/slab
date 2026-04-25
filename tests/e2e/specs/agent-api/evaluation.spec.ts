import { test, expect, u } from '../../fixtures/auth';
import {
	createSite,
	deleteSite,
	createAgentKey,
	agentApi,
	type Site
} from '../../fixtures/data';

test.describe('agent-api: evaluation results', () => {
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

	test('GET /api/agent/evaluation/{buildID} returns array (possibly empty)', async () => {
		const ctx = await agentApi(agentKey);
		try {
			// Trigger a build, wait until terminal, then fetch evaluations.
			const triggered = await ctx.post(u('/api/agent/build'), { data: {} });
			expect(triggered.status()).toBe(202);
			const { build_id } = await triggered.json();

			const start = Date.now();
			const timeoutMs = 120_000;
			let status = '';
			while (Date.now() - start < timeoutMs) {
				const res = await ctx.get(u(`/api/agent/build/${build_id}/status`));
				const body = await res.json();
				status = body.status;
				if (status === 'success' || status === 'failed' || status === 'timeout') break;
				await new Promise((r) => setTimeout(r, 750));
			}

			const res = await ctx.get(u(`/api/agent/evaluation/${build_id}`));
			expect(res.ok()).toBe(true);
			const evals = await res.json();
			// Evaluation rows are produced when build succeeded; if it failed,
			// the array is empty. Either way, the shape is an array.
			expect(Array.isArray(evals)).toBe(true);
			if (status === 'success' && evals.length > 0) {
				const row = evals[0];
				expect(row).toHaveProperty('category');
				expect(row).toHaveProperty('score');
			}
		} finally {
			await ctx.dispose();
		}
	});

	test('GET evaluation for unknown buildID returns array (empty)', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.get(u('/api/agent/evaluation/000000000000000000000000'));
			expect(res.ok()).toBe(true);
			const evals = await res.json();
			expect(Array.isArray(evals)).toBe(true);
			expect(evals.length).toBe(0);
		} finally {
			await ctx.dispose();
		}
	});
});

import { request } from '@playwright/test';
import { test, expect, u } from '../../fixtures/auth';

// Every agent endpoint behind X-Agent-Key middleware must reject anonymous
// access with 401. Mirrors auth-required.spec.ts for the agent surface.

const PROBES: Array<{ method: 'GET' | 'POST' | 'PATCH' | 'DELETE' | 'PUT'; path: string }> = [
	{ method: 'GET', path: '/api/agent/context' },
	{ method: 'POST', path: '/api/agent/pages' },
	{ method: 'PATCH', path: '/api/agent/pages/home' },
	{ method: 'DELETE', path: '/api/agent/pages/home' },
	{ method: 'POST', path: '/api/agent/blocks' },
	{ method: 'POST', path: '/api/agent/pages/home/blocks' },
	{ method: 'PATCH', path: '/api/agent/blocks/some-id' },
	{ method: 'DELETE', path: '/api/agent/blocks/some-id' },
	{ method: 'POST', path: '/api/agent/components' },
	{ method: 'PATCH', path: '/api/agent/components/Hero' },
	{ method: 'POST', path: '/api/agent/css-classes' },
	{ method: 'PATCH', path: '/api/agent/css-classes/btn-primary' },
	{ method: 'PUT', path: '/api/agent/global/header' },
	{ method: 'POST', path: '/api/agent/build' },
	{ method: 'GET', path: '/api/agent/build/some-id/status' },
	{ method: 'GET', path: '/api/agent/evaluation/some-id' },
	{ method: 'GET', path: '/api/agent/media' },
	{ method: 'POST', path: '/api/agent/media' },
	{ method: 'POST', path: '/api/agent/media/from-url' },
	{ method: 'POST', path: '/api/agent/media/from-base64' },
	{ method: 'PATCH', path: '/api/agent/media/some-id' },
	{ method: 'DELETE', path: '/api/agent/media/some-id' }
];

test.describe('security: every agent endpoint requires X-Agent-Key', () => {
	for (const probe of PROBES) {
		test(`${probe.method} ${probe.path} -> 401 without key`, async () => {
			const ctx = await request.newContext();
			try {
				let res;
				const opts = { data: {} };
				switch (probe.method) {
					case 'GET':
						res = await ctx.get(u(probe.path));
						break;
					case 'POST':
						res = await ctx.post(u(probe.path), opts);
						break;
					case 'PATCH':
						res = await ctx.patch(u(probe.path), opts);
						break;
					case 'DELETE':
						res = await ctx.delete(u(probe.path));
						break;
					case 'PUT':
						res = await ctx.put(u(probe.path), opts);
						break;
				}
				expect(res.status()).toBe(401);
			} finally {
				await ctx.dispose();
			}
		});
	}

	test('invalid X-Agent-Key returns 401', async () => {
		const ctx = await request.newContext({
			extraHTTPHeaders: { 'X-Agent-Key': 'ask_definitely_not_a_real_key' }
		});
		try {
			const res = await ctx.get(u('/api/agent/context'));
			expect(res.status()).toBe(401);
		} finally {
			await ctx.dispose();
		}
	});
});

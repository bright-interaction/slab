import { request } from '@playwright/test';
import { test, expect, u } from '../../fixtures/auth';

// Every admin endpoint behind the JWT cookie middleware must answer 401 to
// an anonymous request. We probe a representative slice covering each handler
// family so a future "forgot to mount middleware" regression gets caught.

const PROBES: Array<{ method: 'GET' | 'POST' | 'PATCH' | 'DELETE' | 'PUT'; path: string }> = [
	{ method: 'GET', path: '/api/sites' },
	{ method: 'POST', path: '/api/sites' },
	{ method: 'POST', path: '/api/sites/seed' },
	{ method: 'GET', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa' },
	{ method: 'PATCH', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa' },
	{ method: 'DELETE', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa' },
	{ method: 'GET', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa/silos' },
	{ method: 'GET', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa/pages' },
	{ method: 'POST', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa/pages' },
	{ method: 'GET', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa/components' },
	{ method: 'GET', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa/css-classes' },
	{ method: 'GET', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa/settings' },
	{ method: 'GET', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa/guardrails' },
	{ method: 'GET', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa/profile' },
	{ method: 'GET', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa/allowed-scripts' },
	{ method: 'GET', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa/media' },
	{ method: 'GET', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa/knowledgebase' },
	{ method: 'POST', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa/build' },
	{ method: 'POST', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa/deploy' },
	{ method: 'GET', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa/deploy-targets' },
	{ method: 'POST', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa/deploy-targets' },
	{ method: 'GET', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa/evaluations' },
	{ method: 'GET', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa/agent-keys' },
	{ method: 'POST', path: '/api/sites/aaaaaaaaaaaaaaaaaaaaaaaa/agent-keys' },
	{ method: 'GET', path: '/api/auth/me' },
	{ method: 'POST', path: '/api/auth/change-password' },
	{ method: 'POST', path: '/api/auth/sign-out-everywhere' }
];

test.describe('security: every admin endpoint requires JWT cookie', () => {
	for (const probe of PROBES) {
		test(`${probe.method} ${probe.path} -> 401 without cookie`, async () => {
			const ctx = await request.newContext();
			try {
				let res;
				switch (probe.method) {
					case 'GET':
						res = await ctx.get(u(probe.path));
						break;
					case 'POST':
						res = await ctx.post(u(probe.path), { data: {} });
						break;
					case 'PATCH':
						res = await ctx.patch(u(probe.path), { data: {} });
						break;
					case 'DELETE':
						res = await ctx.delete(u(probe.path));
						break;
					case 'PUT':
						res = await ctx.put(u(probe.path), { data: {} });
						break;
				}
				expect(res.status()).toBe(401);
			} finally {
				await ctx.dispose();
			}
		});
	}
});

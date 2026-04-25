import { test, expect, u } from '../../fixtures/auth';
import {
	createSite,
	deleteSite,
	createAgentKey,
	agentApi,
	type Site
} from '../../fixtures/data';

test.describe('negative: SSRF guard on /api/agent/media/from-url', () => {
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

	const ssrfTargets = [
		{ label: 'AWS metadata IP', url: 'http://169.254.169.254/latest/meta-data/' },
		{ label: 'localhost ssh', url: 'http://127.0.0.1:22' },
		{ label: 'IPv6 loopback', url: 'http://[::1]/' },
		{ label: 'file:// scheme', url: 'file:///etc/passwd' },
		{ label: '10.x private', url: 'http://10.0.0.1/foo.png' },
		{ label: '192.168.x private', url: 'http://192.168.1.1/foo.png' }
	];

	for (const target of ssrfTargets) {
		test(`rejects ${target.label}: ${target.url}`, async () => {
			const ctx = await agentApi(agentKey);
			try {
				const res = await ctx.post(u('/api/agent/media/from-url'), {
					data: { url: target.url, filename: 'x.png' }
				});
				expect(res.status()).toBeGreaterThanOrEqual(400);
				expect(res.status()).toBeLessThan(500);
				const body = await res.json();
				expect(body.error).toBeTruthy();
			} finally {
				await ctx.dispose();
			}
		});
	}

	test('rejects empty URL', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u('/api/agent/media/from-url'), {
				data: { url: '' }
			});
			expect(res.status()).toBe(400);
		} finally {
			await ctx.dispose();
		}
	});

	test('rejects unparseable URL', async () => {
		const ctx = await agentApi(agentKey);
		try {
			const res = await ctx.post(u('/api/agent/media/from-url'), {
				data: { url: 'http://[invalid' }
			});
			expect(res.status()).toBe(400);
		} finally {
			await ctx.dispose();
		}
	});
});

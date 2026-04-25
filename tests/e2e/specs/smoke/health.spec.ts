import { test, expect, u } from '../../fixtures/auth';

test.describe('smoke: server up', () => {
	test('GET /api/health returns ok', async ({ request }) => {
		const res = await request.get(u('/api/health'));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		expect(body.status).toBe('ok');
	});

	test('public starter kits catalog is reachable without auth', async ({ request }) => {
		const res = await request.get(u('/api/starter-kits'));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		const list = body.starter_kits ?? body;
		expect(Array.isArray(list)).toBe(true);
	});

	test('admin route requires auth (no cookie returns 401)', async ({ request }) => {
		const res = await request.get(u('/api/sites'));
		expect(res.status()).toBe(401);
	});

	test('login endpoint rejects bad credentials', async ({ request }) => {
		const res = await request.post(u('/api/auth/login'), {
			data: { email: 'nobody@nowhere.test', password: 'wrong' }
		});
		expect(res.status()).toBe(401);
	});

	test('stubs are up', async ({ playwright, request }) => {
		// CookieProof + CRM stubs are plain http.
		for (const port of [18291, 18292]) {
			const res = await request.get(`http://127.0.0.1:${port}/__health`);
			expect(res.ok()).toBe(true);
		}
		// Dockyard stub serves TLS with a self-signed cert (see fixtures/certs/).
		const tlsCtx = await playwright.request.newContext({ ignoreHTTPSErrors: true });
		const tlsRes = await tlsCtx.get(`https://127.0.0.1:18293/__health`);
		expect(tlsRes.ok()).toBe(true);
		await tlsCtx.dispose();
	});
});

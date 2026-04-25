import { test, expect, u } from '../../fixtures/auth';

test.describe('smoke: admin fixture', () => {
	test('adminApi can list sites', async ({ adminApi }) => {
		const res = await adminApi.get(u('/api/sites'));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		const sites = body.sites ?? body;
		expect(Array.isArray(sites)).toBe(true);
	});

	test('adminApi GET /api/auth/me returns the seeded admin', async ({ adminApi }) => {
		const res = await adminApi.get(u('/api/auth/me'));
		expect(res.ok()).toBe(true);
		const body = await res.json();
		const u_ = body.user ?? body;
		expect(u_.email).toBe('admin@atomicsite.dev');
		expect(u_.role).toBe('admin');
	});

	test('loggedInPage lands on dashboard without UI login', async ({ loggedInPage }) => {
		await loggedInPage.goto('/');
		await loggedInPage.waitForURL((url) => !url.pathname.startsWith('/login'), { timeout: 10_000 });
		expect(loggedInPage.url()).not.toContain('/login');
	});
});

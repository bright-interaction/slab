import { test, expect, u } from '../../fixtures/auth';
import { E2E_CONSTANTS } from '../../playwright.config';

test.describe('auth: login / me / logout', () => {
	test('POST /api/auth/login sets atomicsite_token cookie', async ({ request }) => {
		const res = await request.post(u('/api/auth/login'), {
			data: { email: E2E_CONSTANTS.ADMIN_EMAIL, password: E2E_CONSTANTS.ADMIN_PASSWORD }
		});
		expect(res.ok()).toBe(true);
		const setCookie = res.headersArray().find((h) => h.name.toLowerCase() === 'set-cookie');
		expect(setCookie?.value).toMatch(/atomicsite_token=/);
		expect(setCookie?.value.toLowerCase()).toContain('httponly');
	});

	test('login with empty body returns 400', async ({ request }) => {
		const res = await request.post(u('/api/auth/login'), { data: {} });
		expect(res.status()).toBe(400);
	});

	test('GET /api/auth/me returns 401 without cookie', async ({ request }) => {
		const res = await request.get(u('/api/auth/me'));
		expect(res.status()).toBe(401);
	});

	test('logout clears the cookie', async ({ adminApi }) => {
		const res = await adminApi.post(u('/api/auth/logout'));
		expect(res.ok()).toBe(true);
		const setCookie = res.headersArray().find((h) => h.name.toLowerCase() === 'set-cookie');
		expect(setCookie?.value.toLowerCase()).toMatch(/max-age=0|expires=thu, 01 jan 1970/);
	});

	test('tampered JWT returns 401', async ({ request }) => {
		const tampered = 'eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJhdHRhY2tlciJ9.BAD_SIGNATURE';
		const res = await request.get(u('/api/auth/me'), {
			headers: { Cookie: `atomicsite_token=${tampered}` }
		});
		expect(res.status()).toBe(401);
	});

	test('GET /api/auth/me returns created_at and updated_at', async ({ adminApi }) => {
		const res = await adminApi.get(u('/api/auth/me'));
		expect(res.ok()).toBe(true);
		const body = (await res.json()) as { user: Record<string, unknown> };
		expect(body.user.created_at).toBeTruthy();
		expect(body.user.updated_at).toBeTruthy();
		expect(typeof body.user.created_at).toBe('string');
	});
});

import { request } from '@playwright/test';
import { test, expect, u } from '../../fixtures/auth';

// internal/server/server.go corsMiddleware echoes Access-Control-Allow-Origin
// only for origins that match cfg.BaseURL or localhost in dev. An attacker
// origin (https://attacker.test) must NOT receive an echo header.

test.describe('security: CORS does not echo unknown origins', () => {
	test('attacker origin is not echoed on /api/health', async () => {
		const ctx = await request.newContext({
			extraHTTPHeaders: { Origin: 'https://attacker.test' }
		});
		try {
			const res = await ctx.get(u('/api/health'));
			expect(res.ok()).toBe(true);
			const headers = res.headers();
			const allowOrigin = headers['access-control-allow-origin'];
			if (allowOrigin) {
				expect(allowOrigin).not.toBe('https://attacker.test');
				expect(allowOrigin).not.toBe('*');
			}
		} finally {
			await ctx.dispose();
		}
	});

	test('attacker origin is not echoed on /api/sites (auth endpoint)', async () => {
		const ctx = await request.newContext({
			extraHTTPHeaders: { Origin: 'https://attacker.test' }
		});
		try {
			const res = await ctx.get(u('/api/sites'));
			expect(res.status()).toBe(401);
			const allowOrigin = res.headers()['access-control-allow-origin'];
			if (allowOrigin) {
				expect(allowOrigin).not.toBe('https://attacker.test');
				expect(allowOrigin).not.toBe('*');
			}
		} finally {
			await ctx.dispose();
		}
	});

	test('preflight from attacker origin returns 204 but no allow-origin header', async () => {
		const ctx = await request.newContext({
			extraHTTPHeaders: {
				Origin: 'https://attacker.test',
				'Access-Control-Request-Method': 'POST'
			}
		});
		try {
			const res = await ctx.fetch(u('/api/sites'), { method: 'OPTIONS' });
			expect(res.status()).toBe(204);
			const allowOrigin = res.headers()['access-control-allow-origin'];
			if (allowOrigin) {
				expect(allowOrigin).not.toBe('https://attacker.test');
				expect(allowOrigin).not.toBe('*');
			}
		} finally {
			await ctx.dispose();
		}
	});

	test('configured BASE_URL origin is allowed', async () => {
		const ctx = await request.newContext({
			extraHTTPHeaders: { Origin: 'http://127.0.0.1:18290' }
		});
		try {
			const res = await ctx.get(u('/api/health'));
			expect(res.ok()).toBe(true);
			const allowOrigin = res.headers()['access-control-allow-origin'];
			expect(allowOrigin).toBe('http://127.0.0.1:18290');
		} finally {
			await ctx.dispose();
		}
	});
});
